package main

import (
	"context"
	"log"
	"net/http"

	"finance/backend/internal/config"
	"finance/backend/internal/database"
	apphttp "finance/backend/internal/http"
	"finance/backend/internal/user"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system environment variables")
	}

	cfg := config.Load()

	db, err := database.OpenMySQL(cfg)
	if err != nil {
		log.Fatalf("mysql connection error: %v", err)
	}
	defer db.Close()

	log.Println("MySQL connection OK")

	userRepo := user.NewRepository(db)
	if err := user.EnsureAdmin(context.Background(), userRepo, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		log.Printf("warning: %v", err)
	}

	router, err := apphttp.NewRouter(db, cfg)
	if err != nil {
		log.Fatalf("router init error: %v", err)
	}

	addr := ":" + cfg.AppPort
	log.Printf("Finance backend listening on http://localhost%s", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
