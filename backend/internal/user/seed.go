package user

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// EnsureAdmin creates the first user account from the given credentials if
// the users table is still empty. It is a no-op once at least one user exists.
func EnsureAdmin(ctx context.Context, repo *Repository, username, password string) error {
	count, err := repo.Count(ctx)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	if username == "" || password == "" {
		return fmt.Errorf("aucun utilisateur n'existe et ADMIN_USERNAME/ADMIN_PASSWORD ne sont pas définis")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = repo.Create(ctx, username, "", "", nil, string(hash), nil)
	return err
}
