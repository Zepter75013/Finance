package category

import (
	"context"
	"database/sql"
	"strings"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

const selectColumns = `
	id,
	account_id,
	name,
	type
`

func (r *Repository) List(ctx context.Context, accountID uint64) ([]Category, error) {
	query := `
		SELECT ` + selectColumns + `
		FROM categories
		WHERE account_id = ?
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := make([]Category, 0)

	for rows.Next() {
		var c Category

		if err := rows.Scan(&c.ID, &c.AccountID, &c.Name, &c.Type); err != nil {
			return nil, err
		}

		categories = append(categories, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

func (r *Repository) Create(ctx context.Context, accountID uint64, name string, categoryType Type) (Category, error) {
	query := `
		INSERT INTO categories (account_id, name, type)
		VALUES (?, ?, ?)
	`

	category := Category{
		AccountID: accountID,
		Name:      strings.TrimSpace(name),
		Type:      categoryType,
	}

	result, err := r.db.ExecContext(ctx, query, category.AccountID, category.Name, category.Type)
	if err != nil {
		return Category{}, err
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		return Category{}, err
	}

	selectQuery := `
		SELECT ` + selectColumns + `
		FROM categories
		WHERE id = ?
	`

	err = r.db.QueryRowContext(ctx, selectQuery, lastInsertID).Scan(
		&category.ID,
		&category.AccountID,
		&category.Name,
		&category.Type,
	)
	if err != nil {
		return Category{}, err
	}

	return category, nil
}

func (r *Repository) Update(ctx context.Context, id uint64, name string, categoryType Type) (Category, error) {
	query := `
		UPDATE categories
		SET name = ?, type = ?
		WHERE id = ?
	`

	category := Category{
		ID:   id,
		Name: strings.TrimSpace(name),
		Type: categoryType,
	}

	_, err := r.db.ExecContext(ctx, query, category.Name, category.Type, category.ID)
	if err != nil {
		return Category{}, err
	}

	selectQuery := `
		SELECT ` + selectColumns + `
		FROM categories
		WHERE id = ?
	`

	err = r.db.QueryRowContext(ctx, selectQuery, category.ID).Scan(
		&category.ID,
		&category.AccountID,
		&category.Name,
		&category.Type,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return Category{}, sql.ErrNoRows
		}

		return Category{}, err
	}

	return category, nil
}

func (r *Repository) Delete(ctx context.Context, id uint64) error {
	query := `
		DELETE FROM categories
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, id)
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
