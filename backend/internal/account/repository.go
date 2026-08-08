package account

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// ErrAccountInUse est renvoyée quand la suppression échoue parce que le
// compte est encore référencé par des achats, des revenus ou des catégories
// (contrainte FK ON DELETE RESTRICT) — le handler la traduit en message
// convivial plutôt que de laisser remonter une erreur SQL brute.
var ErrAccountInUse = errors.New("account is still in use")

const mysqlErrForeignKeyConstraint = 1451

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

const selectColumns = `
	id,
	name
`

// Les compteurs (achats, revenus, catégories) permettent au frontend
// d'afficher l'utilisation de chaque compte et de désactiver la suppression
// sans devoir charger en mémoire les achats/revenus de tous les comptes — ces
// listes sont désormais scopées au seul compte actif.
const selectColumnsWithStats = `
	a.id,
	a.name,
	(SELECT COUNT(*) FROM purchases p WHERE p.account_id = a.id) AS purchase_count,
	(SELECT COUNT(*) FROM incomes i WHERE i.account_id = a.id) AS income_count,
	(SELECT COALESCE(SUM(p2.amount), 0) FROM purchases p2 WHERE p2.account_id = a.id) AS total_expense,
	(SELECT COALESCE(SUM(i2.amount), 0) FROM incomes i2 WHERE i2.account_id = a.id) AS total_income,
	(SELECT COUNT(*) FROM categories c WHERE c.account_id = a.id) AS category_count
`

func (r *Repository) List(ctx context.Context) ([]Account, error) {
	query := `
		SELECT ` + selectColumnsWithStats + `
		FROM accounts a
		ORDER BY a.name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := make([]Account, 0)

	for rows.Next() {
		var a Account

		if err := rows.Scan(
			&a.ID,
			&a.Name,
			&a.PurchaseCount,
			&a.IncomeCount,
			&a.TotalExpense,
			&a.TotalIncome,
			&a.CategoryCount,
		); err != nil {
			return nil, err
		}

		accounts = append(accounts, a)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return accounts, nil
}

func (r *Repository) Create(ctx context.Context, name string) (Account, error) {
	query := `
		INSERT INTO accounts (name)
		VALUES (?)
	`

	account := Account{Name: strings.TrimSpace(name)}

	result, err := r.db.ExecContext(ctx, query, account.Name)
	if err != nil {
		return Account{}, err
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		return Account{}, err
	}

	account.ID = uint64(lastInsertID)

	return account, nil
}

func (r *Repository) Update(ctx context.Context, id uint64, name string) (Account, error) {
	query := `
		UPDATE accounts
		SET name = ?
		WHERE id = ?
	`

	account := Account{ID: id, Name: strings.TrimSpace(name)}

	result, err := r.db.ExecContext(ctx, query, account.Name, account.ID)
	if err != nil {
		return Account{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return Account{}, err
	}

	if rowsAffected == 0 {
		if _, err := r.FindByID(ctx, id); err != nil {
			return Account{}, sql.ErrNoRows
		}
	}

	return account, nil
}

func (r *Repository) FindByID(ctx context.Context, id uint64) (Account, error) {
	query := `
		SELECT ` + selectColumns + `
		FROM accounts
		WHERE id = ?
	`

	var a Account
	err := r.db.QueryRowContext(ctx, query, id).Scan(&a.ID, &a.Name)
	if err != nil {
		return Account{}, err
	}

	return a, nil
}

func (r *Repository) Delete(ctx context.Context, id uint64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = ?`, id)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlErrForeignKeyConstraint {
			return ErrAccountInUse
		}

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
