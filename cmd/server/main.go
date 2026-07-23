package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jungli-billa-raj/InjusticeDB/internal/archival"
	"github.com/jungli-billa-raj/InjusticeDB/internal/db"
)

func main() {
	DB_URL := os.Getenv("DB_URL")
	if DB_URL == "" {
		log.Fatal("Enivronment variables don't seem to be set. Couldn't find DB_URL. 🙂")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := db.InitDB(ctx, DB_URL)
	if err != nil {
		log.Fatalf("Error connecting to Database.\nError:%v", err)
	}
	defer pool.Close()

	log.Println("Connected to DB 👍")

	// archiver initialization
	assetRepo := db.NewPostgresAssetRepository(pool)
	var archiver archival.Archiver
	if os.Getenv("ENABLE_WAYBACK") == "true" {
		archiver = archival.NewWaybackArchiver(assetRepo)
	} else {
		archiver = archival.NewNopArchiver()
	}
	_ = archiver
}
