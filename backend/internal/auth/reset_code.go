package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"
)

const ResetCodeDuration = 10 * time.Minute

func generateResetCode() (string, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	n := (uint32(buf[0])<<24 | uint32(buf[1])<<16 | uint32(buf[2])<<8 | uint32(buf[3])) % 1000000

	return fmt.Sprintf("%06d", n), nil
}

// CreateResetCode generates a new 6-digit code for the user, invalidating any
// previously issued code for that user.
func (r *Repository) CreateResetCode(ctx context.Context, userID uint64) (string, error) {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM password_reset_codes WHERE user_id = ?`, userID); err != nil {
		return "", err
	}

	code, err := generateResetCode()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(ResetCodeDuration)

	query := `
		INSERT INTO password_reset_codes (user_id, code, expires_at)
		VALUES (?, ?, ?)
	`

	if _, err := r.db.ExecContext(ctx, query, userID, code, expiresAt); err != nil {
		return "", err
	}

	return code, nil
}

// ConsumeResetCode checks whether the given code is valid and not expired for
// the user, and deletes it (single use) if so.
func (r *Repository) ConsumeResetCode(ctx context.Context, userID uint64, code string) (bool, error) {
	var id uint64

	// Comparé à l'heure passée en paramètre (celle de Go), pas à une fonction
	// "heure actuelle" côté base — voir le commentaire équivalent dans
	// repository.go (FindValidSession) pour le détail du décalage évité.
	query := `
		SELECT id
		FROM password_reset_codes
		WHERE user_id = ? AND code = ? AND expires_at > ?
	`

	err := r.db.QueryRowContext(ctx, query, userID, code, time.Now()).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	if _, err := r.db.ExecContext(ctx, `DELETE FROM password_reset_codes WHERE id = ?`, id); err != nil {
		return false, err
	}

	return true, nil
}
