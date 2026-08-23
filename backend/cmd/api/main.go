package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"finance/backend/internal/config"
	"finance/backend/internal/database"
	apphttp "finance/backend/internal/http"
	"finance/backend/internal/recurring"
	"finance/backend/internal/user"

	"github.com/joho/godotenv"
)

// runRecurringScheduler exécute les transactions récurrentes dues, une fois
// immédiatement (rattrape ce qui a pu être manqué pendant un arrêt du
// conteneur), puis toutes les heures — largement suffisant pour une
// granularité "jour du mois", inutile d'introduire une dépendance de cron.
func runRecurringScheduler(recurringService *recurring.Service) {
	ctx := context.Background()

	recurringService.RunDue(ctx, time.Now())

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		recurringService.RunDue(ctx, time.Now())
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system environment variables")
	}

	cfg := config.Load()

	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("database connection error (driver=%s): %v", cfg.DBDriver, err)
	}
	defer db.Close()

	log.Printf("Database connection OK (driver=%s)", cfg.DBDriver)

	userRepo := user.NewRepository(db)
	if err := user.EnsureAdmin(context.Background(), userRepo, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		log.Printf("warning: %v", err)
	}

	router, recurringService, err := apphttp.NewRouter(db, cfg)
	if err != nil {
		log.Fatalf("router init error: %v", err)
	}

	go runRecurringScheduler(recurringService)

	addr := ":" + cfg.AppPort
	log.Printf("Finance backend listening on http://localhost%s", addr)

	// http.ListenAndServe seul n'impose aucun délai — une connexion bloquée
	// (écriture disque lente, client qui n'envoie jamais la fin du corps...)
	// retiendrait sa goroutine indéfiniment. 5 minutes reste généreux pour
	// les plus gros transferts de l'app (upload de sauvegarde SQL, rapport
	// PDF) tout en garantissant qu'une requête finit toujours par échouer
	// proprement plutôt que de bloquer pour toujours.
	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       2 * time.Minute,
		ReadHeaderTimeout: 10 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
