package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jungli-billa-raj/InjusticeDB/internal/archival"
	"github.com/jungli-billa-raj/InjusticeDB/internal/db"
)

type Server struct {
	router   *chi.Mux
	repos    *db.Repositories
	archiver archival.Archiver
	cfg      Config
}

func NewServer(repos *db.Repositories, archiver archival.Archiver, cfg Config) *Server {
	r := chi.NewRouter()

	// Standard Middlewares
	r.Use(middleware.RequestID)
	// r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS Setup for Flutter (Web, Android, iOS)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	s := &Server{
		router:   r,
		repos:    repos,
		archiver: archiver,
		cfg:      cfg,
	}

	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) routes() {
	// Health Check Endpoint
	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		JSON(w, http.StatusOK, map[string]string{"status": "healthy"})
	})

	// API v1 Sub-Router
	s.router.Route("/api/v1", func(r chi.Router) {
		// Public Auth Routes
		r.Post("/auth/google", s.HandleGoogleLogin)

		// Additional routes will be attached here
	})
}
