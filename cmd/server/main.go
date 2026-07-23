package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jungli-billa-raj/InjusticeDB/internal/api"
	"github.com/jungli-billa-raj/InjusticeDB/internal/archival"
	"github.com/jungli-billa-raj/InjusticeDB/internal/db"
)

func main() {
	// 1. Read Environment Variables
	dbURL := getEnv("DB_URL", "postgres://postgres:postgres@localhost:5432/injusticedb?sslmode=disable")
	port := getEnv("PORT", "8080")
	jwtSecret := getEnv("JWT_SECRET", "super-secret-key-change-in-production")
	googleClientID := getEnv("GOOGLE_CLIENT_ID", "")
	enableWayback := getEnv("ENABLE_WAYBACK", "false") == "true"

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. Initialize PostgreSQL Connection Pool
	log.Println("[INFO] Connecting to PostgreSQL database...")
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("[FATAL] Unable to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("[FATAL] Database ping failed: %v", err)
	}
	log.Println("[INFO] Connected to PostgreSQL successfully.")

	// 3. Initialize Concrete Repositories
	repos := &db.Repositories{
		Users:         db.NewPostgresUserRepository(pool),
		Incidents:     db.NewPostgresIncidentRepository(pool),
		Culprits:      db.NewPostgresCulpritRepository(pool),
		Assets:        db.NewPostgresAssetRepository(pool),
		Verifications: db.NewPostgresVerificationRepository(pool),
		Messaging:     db.NewPostgresMessagingRepository(pool),
		Comments:      db.NewPostgresCommentRepository(pool),
		Targets:       db.NewPostgresTargetRepository(pool),
	}

	// 4. Initialize Wayback Archiver Service
	var archiver archival.Archiver
	if enableWayback {
		log.Println("[INFO] Wayback Machine Archiver ENABLED (Production mode).")
		archiver = archival.NewWaybackArchiver(repos.Assets)
	} else {
		log.Println("[INFO] Wayback Machine Archiver DISABLED (Dry-run mode).")
		archiver = archival.NewNopArchiver()
	}

	// 5. Initialize API Server
	cfg := api.Config{
		JWTSecret:      jwtSecret,
		GoogleClientID: googleClientID,
		EnableWayback:  enableWayback,
	}

	server := api.NewServer(repos, archiver, cfg)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      server,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 6. Start HTTP Server in a Background Goroutine
	go func() {
		log.Printf("[INFO] Server listening on port %s...", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] HTTP server error: %v", err)
		}
	}()

	// 7. Graceful Shutdown listener
	<-ctx.Done()
	log.Println("[INFO] Shutting down server gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("[FATAL] Forced server shutdown error: %v", err)
	}

	log.Println("[INFO] Server stopped safely.")
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}
