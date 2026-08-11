package subcategory

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

func (r *Repository) List(ctx context.Context) ([]SubCategory, error) {
	query := `
		SELECT id, category_id, name, created_at, updated_at
		FROM sub_categories
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subCategories := make([]SubCategory, 0)

	for rows.Next() {
		var sc SubCategory

		if err := rows.Scan(&sc.ID, &sc.CategoryID, &sc.Name, &sc.CreatedAt, &sc.UpdatedAt); err != nil {
			return nil, err
		}

		subCategories = append(subCategories, sc)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subCategories, nil
}

func (r *Repository) FindByCategoryAndName(ctx context.Context, categoryID uint64, name string) (SubCategory, error) {
	query := `
		SELECT id, category_id, name, created_at, updated_at
		FROM sub_categories
		WHERE category_id = ? AND name = ?
	`

	var sc SubCategory

	err := r.db.QueryRowContext(ctx, query, categoryID, name).Scan(
		&sc.ID, &sc.CategoryID, &sc.Name, &sc.CreatedAt, &sc.UpdatedAt,
	)
	if err != nil {
		return SubCategory{}, err
	}

	return sc, nil
}

func (r *Repository) Create(ctx context.Context, categoryID uint64, name string) (SubCategory, error) {
	query := `
		INSERT INTO sub_categories (category_id, name)
		VALUES (?, ?)
	`

	name = strings.TrimSpace(name)

	result, err := r.db.ExecContext(ctx, query, categoryID, name)
	if err != nil {
		return SubCategory{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return SubCategory{}, err
	}

	return r.FindByID(ctx, uint64(id))
}

func (r *Repository) FindByID(ctx context.Context, id uint64) (SubCategory, error) {
	query := `
		SELECT id, category_id, name, created_at, updated_at
		FROM sub_categories
		WHERE id = ?
	`

	var sc SubCategory

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&sc.ID, &sc.CategoryID, &sc.Name, &sc.CreatedAt, &sc.UpdatedAt,
	)
	if err != nil {
		return SubCategory{}, err
	}

	return sc, nil
}

// Update renames a sub-category and cascades the new name to every purchase
// and income row currently using the old one — purchases/incomes store the
// sub-category as a plain string rather than a foreign key, so without this
// cascade a rename would silently orphan existing transactions.
func (r *Repository) Update(ctx context.Context, id uint64, name string) (SubCategory, error) {
	name = strings.TrimSpace(name)

	current, err := r.FindByID(ctx, id)
	if err != nil {
		return SubCategory{}, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return SubCategory{}, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE sub_categories SET name = ? WHERE id = ?`, name, id); err != nil {
		return SubCategory{}, err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE purchases SET sub_category = ? WHERE category_id = ? AND sub_category = ?`,
		name, current.CategoryID, current.Name,
	); err != nil {
		return SubCategory{}, err
	}

	var categoryName string
	if err := tx.QueryRowContext(ctx, `SELECT name FROM categories WHERE id = ?`, current.CategoryID).Scan(&categoryName); err != nil {
		return SubCategory{}, err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE incomes SET sub_category = ? WHERE category = ? AND sub_category = ?`,
		name, categoryName, current.Name,
	); err != nil {
		return SubCategory{}, err
	}

	if err := tx.Commit(); err != nil {
		return SubCategory{}, err
	}

	return r.FindByID(ctx, id)
}

// AccountIDOfCategory résout le compte d'une catégorie — utilisé pour
// vérifier les droits d'accès sur Create, qui ne reçoit que category_id.
func (r *Repository) AccountIDOfCategory(ctx context.Context, categoryID uint64) (uint64, error) {
	var accountID uint64

	err := r.db.QueryRowContext(ctx, `SELECT account_id FROM categories WHERE id = ?`, categoryID).Scan(&accountID)
	if err != nil {
		return 0, err
	}

	return accountID, nil
}

// AccountIDOf résout le compte d'une sous-catégorie existante (via sa
// catégorie) — utilisé pour vérifier les droits d'accès sur Update/Delete.
func (r *Repository) AccountIDOf(ctx context.Context, id uint64) (uint64, error) {
	var accountID uint64

	query := `
		SELECT c.account_id
		FROM sub_categories sc
		JOIN categories c ON c.id = sc.category_id
		WHERE sc.id = ?
	`

	err := r.db.QueryRowContext(ctx, query, id).Scan(&accountID)
	if err != nil {
		return 0, err
	}

	return accountID, nil
}

func (r *Repository) Delete(ctx context.Context, id uint64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM sub_categories WHERE id = ?`, id)
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
