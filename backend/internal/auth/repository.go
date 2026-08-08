package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"
)

const SessionDuration = 30 * 24 * time.Hour

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

func (r *Repository) CreateSession(ctx context.Context, userID uint64) (Session, error) {
	token, err := generateToken()
	if err != nil {
		return Session{}, err
	}

	expiresAt := time.Now().Add(SessionDuration)

	query := `
		INSERT INTO sessions (token, user_id, expires_at)
		VALUES (?, ?, ?)
	`

	if _, err := r.db.ExecContext(ctx, query, token, userID, expiresAt); err != nil {
		return Session{}, err
	}

	return Session{
		Token:     token,
		UserID:    userID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}, nil
}

func (r *Repository) FindValidSession(ctx context.Context, token string) (Session, error) {
	query := `
		SELECT token, user_id, expires_at, created_at
		FROM sessions
		WHERE token = ? AND expires_at > NOW()
	`

	var s Session

	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&s.Token,
		&s.UserID,
		&s.ExpiresAt,
		&s.CreatedAt,
	)
	if err != nil {
		return Session{}, err
	}

	return s, nil
}

func (r *Repository) DeleteSession(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, token)
	return err
}
