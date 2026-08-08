package user

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

func (r *Repository) Count(ctx context.Context) (int, error) {
	var count int

	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *Repository) List(ctx context.Context) ([]User, error) {
	query := `
		SELECT id, username, first_name, last_name, avatar_url, password_hash, created_at, updated_at
		FROM users
		ORDER BY username ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]User, 0)

	for rows.Next() {
		var u User

		if err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.FirstName,
			&u.LastName,
			&u.AvatarURL,
			&u.PasswordHash,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, err
		}

		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *Repository) Delete(ctx context.Context, id uint64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *Repository) FindByUsername(ctx context.Context, username string) (User, error) {
	query := `
		SELECT id, username, first_name, last_name, avatar_url, password_hash, created_at, updated_at
		FROM users
		WHERE username = ?
	`

	var u User

	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&u.ID,
		&u.Username,
		&u.FirstName,
		&u.LastName,
		&u.AvatarURL,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return User{}, err
	}

	return u, nil
}

func (r *Repository) Create(ctx context.Context, username, firstName, lastName string, avatarURL *string, passwordHash string) (User, error) {
	query := `
		INSERT INTO users (username, first_name, last_name, avatar_url, password_hash)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query, username, firstName, lastName, avatarURL, passwordHash)
	if err != nil {
		return User{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return User{}, err
	}

	return r.FindByID(ctx, uint64(id))
}

func (r *Repository) Update(ctx context.Context, id uint64, username, firstName, lastName string, avatarURL *string) (User, error) {
	query := `
		UPDATE users
		SET username = ?, first_name = ?, last_name = ?, avatar_url = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, username, firstName, lastName, avatarURL, id)
	if err != nil {
		return User{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return User{}, err
	}

	if rowsAffected == 0 {
		if _, err := r.FindByID(ctx, id); err != nil {
			return User{}, sql.ErrNoRows
		}
	}

	return r.FindByID(ctx, id)
}

func (r *Repository) UpdatePassword(ctx context.Context, id uint64, passwordHash string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *Repository) FindByID(ctx context.Context, id uint64) (User, error) {
	query := `
		SELECT id, username, first_name, last_name, avatar_url, password_hash, created_at, updated_at
		FROM users
		WHERE id = ?
	`

	var u User

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.Username,
		&u.FirstName,
		&u.LastName,
		&u.AvatarURL,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return User{}, err
	}

	return u, nil
}
