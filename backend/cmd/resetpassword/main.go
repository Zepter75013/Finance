// Command resetpassword sets (or creates) a user's password directly in the
// database. Use it when you're locked out of the app and don't know your
// current password — it requires local terminal access to the server/DB,
// the same trust level as reading the .env file.
//
// Usage:
//
//	go run ./cmd/resetpassword -username you@example.com -password 'new-password'
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"

	"finance/backend/internal/config"
	"finance/backend/internal/database"
	"finance/backend/internal/user"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	username := flag.String("username", "", "identifiant du compte à créer/réinitialiser")
	password := flag.String("password", "", "nouveau mot de passe")
	flag.Parse()

	if *username == "" || *password == "" {
		log.Fatal("usage: go run ./cmd/resetpassword -username <identifiant> -password <nouveau mot de passe>")
	}

	if len(*password) < 4 {
		log.Fatal("le mot de passe doit contenir au moins 4 caractères")
	}

	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system environment variables")
	}

	cfg := config.Load()

	db, err := database.OpenMySQL(cfg)
	if err != nil {
		log.Fatalf("mysql connection error: %v", err)
	}
	defer db.Close()

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("password hashing error: %v", err)
	}

	repo := user.NewRepository(db)
	ctx := context.Background()

	account, err := repo.FindByUsername(ctx, *username)
	if err == sql.ErrNoRows {
		if _, err := repo.Create(ctx, *username, "", "", nil, string(hash), nil, true); err != nil {
			log.Fatalf("failed to create user: %v", err)
		}

		fmt.Printf("Compte créé pour %q avec le nouveau mot de passe.\n", *username)
		return
	}

	if err != nil {
		log.Fatalf("failed to look up user: %v", err)
	}

	if err := repo.UpdatePassword(ctx, account.ID, string(hash)); err != nil {
		log.Fatalf("failed to update password: %v", err)
	}

	fmt.Printf("Mot de passe réinitialisé pour %q.\n", *username)
}
