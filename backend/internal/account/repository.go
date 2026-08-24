package account

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

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

// Les compteurs (achats, revenus, catégories) permettent au frontend
// d'afficher l'utilisation de chaque compte et de désactiver la suppression
// sans devoir charger en mémoire les achats/revenus de tous les comptes — ces
// listes sont désormais scopées au seul compte actif.
const selectColumnsWithStats = `
	a.id,
	a.name,
	a.opening_balance_amount,
	a.opening_balance_date,
	a.has_statements,
	(SELECT COUNT(*) FROM purchases p WHERE p.account_id = a.id) AS purchase_count,
	(SELECT COUNT(*) FROM incomes i WHERE i.account_id = a.id) AS income_count,
	(SELECT COALESCE(SUM(p2.amount), 0) FROM purchases p2 WHERE p2.account_id = a.id) AS total_expense,
	(SELECT COALESCE(SUM(i2.amount), 0) FROM incomes i2 WHERE i2.account_id = a.id) AS total_income,
	(SELECT COUNT(*) FROM categories c WHERE c.account_id = a.id) AS category_count,
	(SELECT COUNT(*) FROM transfers t WHERE t.from_account_id = a.id OR t.to_account_id = a.id) AS transfer_count
`

// scanOpeningBalance convertit les colonnes nullable opening_balance_amount/
// date scannées dans des types intermédiaires vers les pointeurs exposés par
// Account — même convention que purchase.Repository pour operation_date.
func scanOpeningBalance(amount sql.NullFloat64, date sql.NullTime) (*float64, *string) {
	var amountPtr *float64
	var datePtr *string

	if amount.Valid {
		v := amount.Float64
		amountPtr = &v
	}

	if date.Valid {
		formatted := date.Time.Format("2006-01-02")
		datePtr = &formatted
	}

	return amountPtr, datePtr
}

// List renvoie les comptes visibles par userID : tous les comptes si cet
// utilisateur n'a jamais été restreint (users.accounts_restricted = 0),
// sinon uniquement ceux qui lui sont explicitement assignés dans
// user_accounts (potentiellement aucun).
func (r *Repository) List(ctx context.Context, userID uint64) ([]Account, error) {
	query := `
		SELECT ` + selectColumnsWithStats + `
		FROM accounts a
		WHERE (SELECT accounts_restricted FROM users WHERE id = ?) = 0
		   OR EXISTS (SELECT 1 FROM user_accounts ua WHERE ua.user_id = ? AND ua.account_id = a.id)
		ORDER BY a.name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := make([]Account, 0)

	for rows.Next() {
		var a Account
		var openingAmount sql.NullFloat64
		var openingDate sql.NullTime

		if err := rows.Scan(
			&a.ID,
			&a.Name,
			&openingAmount,
			&openingDate,
			&a.HasStatements,
			&a.PurchaseCount,
			&a.IncomeCount,
			&a.TotalExpense,
			&a.TotalIncome,
			&a.CategoryCount,
			&a.TransferCount,
		); err != nil {
			return nil, err
		}

		a.OpeningBalanceAmount, a.OpeningBalanceDate = scanOpeningBalance(openingAmount, openingDate)

		accounts = append(accounts, a)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return accounts, nil
}

// UserCanAccess indique si userID peut agir sur accountID : vrai si cet
// utilisateur n'a jamais été restreint (users.accounts_restricted = 0), ou
// s'il a explicitement ce compte parmi les siens.
func (r *Repository) UserCanAccess(ctx context.Context, userID, accountID uint64) (bool, error) {
	query := `
		SELECT (SELECT accounts_restricted FROM users WHERE id = ?) = 0
		    OR EXISTS (SELECT 1 FROM user_accounts WHERE user_id = ? AND account_id = ?)
	`

	var allowed bool
	if err := r.db.QueryRowContext(ctx, query, userID, userID, accountID).Scan(&allowed); err != nil {
		return false, err
	}

	return allowed, nil
}

func (r *Repository) Create(ctx context.Context, name string) (Account, error) {
	query := `
		INSERT INTO accounts (name)
		VALUES (?)
	`

	account := Account{Name: strings.TrimSpace(name), HasStatements: true}

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

// SetOpeningBalance déclare un solde de départ à une date donnée — sert
// d'ancrage à computeRealBalance (frontend) quand ce compte n'a aucun relevé
// verrouillé (compte type Livret, qui n'en a structurellement jamais).
func (r *Repository) SetOpeningBalance(ctx context.Context, id uint64, amount float64, date string) (Account, error) {
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return Account{}, err
	}

	result, err := r.db.ExecContext(ctx,
		`UPDATE accounts SET opening_balance_amount = ?, opening_balance_date = ? WHERE id = ?`,
		amount, parsedDate, id,
	)
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

	return r.FindByID(ctx, id)
}

// ClearOpeningBalance efface le solde de départ déclaré — le compte retombe
// alors sur le repli "mouvements non rapprochés" sans ancrage.
func (r *Repository) ClearOpeningBalance(ctx context.Context, id uint64) (Account, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE accounts SET opening_balance_amount = NULL, opening_balance_date = NULL WHERE id = ?`,
		id,
	)
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

	return r.FindByID(ctx, id)
}

// SetHasStatements active/désactive la notion de relevé bancaire pour ce
// compte — désactivée, l'écran Pointage n'a plus d'utilité pour ce compte
// (ex: Livret, qui n'a structurellement jamais de relevé).
func (r *Repository) SetHasStatements(ctx context.Context, id uint64, hasStatements bool) (Account, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE accounts SET has_statements = ? WHERE id = ?`,
		hasStatements, id,
	)
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

	return r.FindByID(ctx, id)
}

func (r *Repository) Update(ctx context.Context, id uint64, name string) (Account, error) {
	query := `
		UPDATE accounts
		SET name = ?
		WHERE id = ?
	`

	trimmedName := strings.TrimSpace(name)

	result, err := r.db.ExecContext(ctx, query, trimmedName, id)
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

	// Re-charge la ligne complète plutôt que de renvoyer un Account partiel
	// (id/name seulement) — sinon solde initial, has_statements, etc.
	// seraient effacés côté client au prochain remplacement de la liste.
	return r.FindByID(ctx, id)
}

// FindByID utilise selectColumnsWithStats (comme List) et non selectColumns
// — un appelant après mutation (SetOpeningBalance, SetHasStatements, Update…)
// remplace tout l'objet Account côté frontend avec ce résultat ; sans les
// compteurs, achats/revenus/catégories/virements retombaient à zéro dans
// l'UI jusqu'au prochain rechargement complet de la page.
func (r *Repository) FindByID(ctx context.Context, id uint64) (Account, error) {
	query := `
		SELECT ` + selectColumnsWithStats + `
		FROM accounts a
		WHERE a.id = ?
	`

	var a Account
	var openingAmount sql.NullFloat64
	var openingDate sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&a.ID,
		&a.Name,
		&openingAmount,
		&openingDate,
		&a.HasStatements,
		&a.PurchaseCount,
		&a.IncomeCount,
		&a.TotalExpense,
		&a.TotalIncome,
		&a.CategoryCount,
		&a.TransferCount,
	)
	if err != nil {
		return Account{}, err
	}

	a.OpeningBalanceAmount, a.OpeningBalanceDate = scanOpeningBalance(openingAmount, openingDate)

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
