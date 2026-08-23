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

func (r *Repository) CountAdmins(ctx context.Context) (int, error) {
	var count int

	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE is_admin = 1`).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// accountIDsOf charge les comptes assignés à un utilisateur — une requête
// par utilisateur, volontairement simple vu le faible nombre d'utilisateurs
// de cette application.
func (r *Repository) accountIDsOf(ctx context.Context, userID uint64) ([]uint64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT account_id FROM user_accounts WHERE user_id = ? ORDER BY account_id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accountIDs := make([]uint64, 0)

	for rows.Next() {
		var accountID uint64
		if err := rows.Scan(&accountID); err != nil {
			return nil, err
		}

		accountIDs = append(accountIDs, accountID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return accountIDs, nil
}

func (r *Repository) List(ctx context.Context) ([]User, error) {
	query := `
		SELECT id, username, first_name, last_name, avatar_url, password_hash, accounts_restricted, is_admin, created_at, updated_at
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
		var restricted bool

		if err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.FirstName,
			&u.LastName,
			&u.AvatarURL,
			&u.PasswordHash,
			&restricted,
			&u.IsAdmin,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if restricted {
			accountIDs, err := r.accountIDsOf(ctx, u.ID)
			if err != nil {
				return nil, err
			}

			u.AccountIDs = accountIDs
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
		SELECT id, username, first_name, last_name, avatar_url, password_hash, accounts_restricted, is_admin, created_at, updated_at
		FROM users
		WHERE username = ?
	`

	var u User
	var restricted bool

	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&u.ID,
		&u.Username,
		&u.FirstName,
		&u.LastName,
		&u.AvatarURL,
		&u.PasswordHash,
		&restricted,
		&u.IsAdmin,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return User{}, err
	}

	if restricted {
		accountIDs, err := r.accountIDsOf(ctx, u.ID)
		if err != nil {
			return User{}, err
		}

		u.AccountIDs = accountIDs
	}

	return u, nil
}

// setAccountAssignments remplace la liste des comptes assignés à un
// utilisateur et marque users.accounts_restricted en conséquence.
// accountIDs à nil signifie "ne pas toucher à la restriction" (créer
// l'utilisateur sans en parler laisse l'accès à tous les comptes, valeur par
// défaut) ; une slice non nil (même vide) marque explicitement l'utilisateur
// comme restreint.
func (r *Repository) setAccountAssignments(ctx context.Context, userID uint64, accountIDs *[]uint64) error {
	if accountIDs == nil {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE users SET accounts_restricted = 1 WHERE id = ?`, userID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM user_accounts WHERE user_id = ?`, userID); err != nil {
		return err
	}

	for _, accountID := range *accountIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO user_accounts (user_id, account_id) VALUES (?, ?)`,
			userID, accountID,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) Create(ctx context.Context, username, firstName, lastName string, avatarURL *string, passwordHash string, accountIDs *[]uint64, isAdmin bool) (User, error) {
	query := `
		INSERT INTO users (username, first_name, last_name, avatar_url, password_hash, is_admin)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query, username, firstName, lastName, avatarURL, passwordHash, isAdmin)
	if err != nil {
		return User{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return User{}, err
	}

	if err := r.setAccountAssignments(ctx, uint64(id), accountIDs); err != nil {
		return User{}, err
	}

	return r.FindByID(ctx, uint64(id))
}

// Update modifie le profil d'un utilisateur. isAdmin à nil laisse le statut
// admin inchangé (cas d'une auto-édition de profil par un non-admin, qui ne
// doit jamais pouvoir s'accorder ce privilège lui-même).
func (r *Repository) Update(ctx context.Context, id uint64, username, firstName, lastName string, avatarURL *string, accountIDs *[]uint64, isAdmin *bool) (User, error) {
	query := `
		UPDATE users
		SET username = ?, first_name = ?, last_name = ?, avatar_url = ?
		WHERE id = ?
	`
	args := []any{username, firstName, lastName, avatarURL, id}

	if isAdmin != nil {
		query = `
			UPDATE users
			SET username = ?, first_name = ?, last_name = ?, avatar_url = ?, is_admin = ?
			WHERE id = ?
		`
		args = []any{username, firstName, lastName, avatarURL, *isAdmin, id}
	}

	result, err := r.db.ExecContext(ctx, query, args...)
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

	if err := r.setAccountAssignments(ctx, id, accountIDs); err != nil {
		return User{}, err
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
		SELECT id, username, first_name, last_name, avatar_url, password_hash, accounts_restricted, is_admin, created_at, updated_at
		FROM users
		WHERE id = ?
	`

	var u User
	var restricted bool

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.Username,
		&u.FirstName,
		&u.LastName,
		&u.AvatarURL,
		&u.PasswordHash,
		&restricted,
		&u.IsAdmin,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return User{}, err
	}

	if restricted {
		accountIDs, err := r.accountIDsOf(ctx, u.ID)
		if err != nil {
			return User{}, err
		}

		u.AccountIDs = accountIDs
	}

	return u, nil
}
