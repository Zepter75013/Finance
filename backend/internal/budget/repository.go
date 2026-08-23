package budget

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

// GetForAccount renvoie les budgets ajustés explicitement pour ce compte et
// ce mois, sous forme de map category_id -> montant — même forme que
// l'objet categoryBudgetOverrides côté frontend. Les catégories sans ligne
// ici n'ont pas d'ajustement explicite (le budget effectif retombe alors sur
// la moyenne suggérée, calculée par Service.EffectiveBudgets).
func (r *Repository) GetForAccount(ctx context.Context, accountID uint64, monthKey string) (map[uint64]float64, error) {
	query := `
		SELECT cb.category_id, cb.amount
		FROM category_budgets cb
		JOIN categories c ON c.id = cb.category_id
		WHERE c.account_id = ? AND cb.month_key = ?
	`

	rows, err := r.db.QueryContext(ctx, query, accountID, monthKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uint64]float64)

	for rows.Next() {
		var categoryID uint64
		var amount float64

		if err := rows.Scan(&categoryID, &amount); err != nil {
			return nil, err
		}

		result[categoryID] = amount
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// Upsert crée ou remplace l'ajustement de budget d'une catégorie pour un
// mois donné. Portable entre MySQL et SQLite (pas de syntaxe UPSERT native
// utilisée ailleurs dans ce projet) : tente d'abord une mise à jour, insère
// seulement si aucune ligne n'existait déjà.
func (r *Repository) Upsert(ctx context.Context, categoryID uint64, monthKey string, amount float64) (CategoryBudget, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE category_budgets SET amount = ? WHERE category_id = ? AND month_key = ?`,
		amount, categoryID, monthKey,
	)
	if err != nil {
		return CategoryBudget{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return CategoryBudget{}, err
	}

	if rowsAffected == 0 {
		if _, err := r.db.ExecContext(ctx,
			`INSERT INTO category_budgets (category_id, month_key, amount) VALUES (?, ?, ?)`,
			categoryID, monthKey, amount,
		); err != nil {
			return CategoryBudget{}, err
		}
	}

	var budget CategoryBudget
	err = r.db.QueryRowContext(ctx,
		`SELECT id, category_id, month_key, amount, created_at, updated_at FROM category_budgets WHERE category_id = ? AND month_key = ?`,
		categoryID, monthKey,
	).Scan(&budget.ID, &budget.CategoryID, &budget.MonthKey, &budget.Amount, &budget.CreatedAt, &budget.UpdatedAt)
	if err != nil {
		return CategoryBudget{}, err
	}

	return budget, nil
}

func (r *Repository) Delete(ctx context.Context, categoryID uint64, monthKey string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM category_budgets WHERE category_id = ? AND month_key = ?`,
		categoryID, monthKey,
	)

	return err
}
