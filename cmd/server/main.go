package main

import (
	"fmt"
	"go-music-tag/config"
	"go-music-tag/database"
	"go-music-tag/routes"
	"log"
	"net/http"
	"time"
)

func main() {
	cfg := config.GetConfig()

	if err := database.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	r := routes.SetupRouter()

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("========================================")
	log.Printf("Music Tag Manager Starting...")
	log.Printf("========================================")
	log.Printf("Server: http://localhost%s", addr)
	log.Printf("Mode: %s", cfg.Server.Mode)
	log.Printf("Database: %s", cfg.Database.Path)
	log.Printf("========================================")

	s := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if err := s.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
