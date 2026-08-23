package auditlog

import (
	"context"
	"database/sql"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, entry Entry) error {
	var userID sql.NullInt64
	if entry.UserID != nil {
		userID = sql.NullInt64{Int64: int64(*entry.UserID), Valid: true}
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO audit_log (user_id, username, method, path, entity_type, entity_id, status_code)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, entry.Username, entry.Method, entry.Path, entry.EntityType, entry.EntityID, entry.StatusCode)

	return err
}

// ListRecent renvoie les entrées les plus récentes d'abord — pas de
// pagination pour ce premier jet, une limite fixe suffit pour un usage
// personnel/petite équipe.
func (r *Repository) ListRecent(ctx context.Context, limit int) ([]Entry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, username, method, path, entity_type, entity_id, status_code, created_at
		FROM audit_log
		ORDER BY created_at DESC, id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]Entry, 0)

	for rows.Next() {
		var entry Entry
		var userID sql.NullInt64

		if err := rows.Scan(
			&entry.ID,
			&userID,
			&entry.Username,
			&entry.Method,
			&entry.Path,
			&entry.EntityType,
			&entry.EntityID,
			&entry.StatusCode,
			&entry.CreatedAt,
		); err != nil {
			return nil, err
		}

		if userID.Valid {
			id := uint64(userID.Int64)
			entry.UserID = &id
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}
