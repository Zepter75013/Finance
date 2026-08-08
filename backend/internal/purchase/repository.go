package purchase

import (
	"context"
	"database/sql"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

const selectColumns = `
	id,
	merchant,
	payment_method,
	category_id,
	account_id,
	amount,
	purchase_date,
	note,
	reference,
	operation_label,
	additional_info,
	sub_category,
	operation_date,
	value_date,
	is_reconciled,
	statement_reference,
	created_at,
	updated_at
`

func scanPurchase(scanner interface {
	Scan(dest ...any) error
}) (Purchase, error) {
	var p Purchase
	var note sql.NullString
	var additionalInfo sql.NullString
	var operationDate sql.NullTime
	var valueDate sql.NullTime

	err := scanner.Scan(
		&p.ID,
		&p.Merchant,
		&p.PaymentMethod,
		&p.CategoryID,
		&p.AccountID,
		&p.Amount,
		&p.PurchaseDate,
		&note,
		&p.Reference,
		&p.OperationLabel,
		&additionalInfo,
		&p.SubCategory,
		&operationDate,
		&valueDate,
		&p.IsReconciled,
		&p.StatementReference,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		return Purchase{}, err
	}

	if note.Valid {
		p.Note = note.String
	}

	if additionalInfo.Valid {
		p.AdditionalInfo = additionalInfo.String
	}

	if operationDate.Valid {
		formatted := operationDate.Time.Format("2006-01-02")
		p.OperationDate = &formatted
	}

	if valueDate.Valid {
		formatted := valueDate.Time.Format("2006-01-02")
		p.ValueDate = &formatted
	}

	return p, nil
}

func (r *Repository) List(ctx context.Context, accountID uint64) ([]Purchase, error) {
	query := `SELECT ` + selectColumns + ` FROM purchases WHERE account_id = ? ORDER BY purchase_date DESC, id DESC`

	rows, err := r.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	purchases := make([]Purchase, 0)

	for rows.Next() {
		p, err := scanPurchase(rows)
		if err != nil {
			return nil, err
		}

		purchases = append(purchases, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return purchases, nil
}

func nullableDate(value string) (sql.NullTime, error) {
	if value == "" {
		return sql.NullTime{}, nil
	}

	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return sql.NullTime{}, err
	}

	return sql.NullTime{Time: parsed, Valid: true}, nil
}

func (r *Repository) Create(ctx context.Context, input CreatePurchaseInput) (Purchase, error) {
	purchaseDate, err := time.Parse("2006-01-02", input.PurchaseDate)
	if err != nil {
		return Purchase{}, err
	}

	operationDate, err := nullableDate(input.OperationDate)
	if err != nil {
		return Purchase{}, err
	}

	valueDate, err := nullableDate(input.ValueDate)
	if err != nil {
		return Purchase{}, err
	}

	insertQuery := `
		INSERT INTO purchases (
			merchant,
			payment_method,
			category_id,
			account_id,
			amount,
			purchase_date,
			note,
			reference,
			operation_label,
			additional_info,
			sub_category,
			operation_date,
			value_date,
			is_reconciled,
			statement_reference
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(
		ctx,
		insertQuery,
		input.Merchant,
		input.PaymentMethod,
		input.CategoryID,
		input.AccountID,
		input.Amount,
		purchaseDate,
		input.Note,
		input.Reference,
		input.OperationLabel,
		input.AdditionalInfo,
		input.SubCategory,
		operationDate,
		valueDate,
		input.IsReconciled,
		input.StatementReference,
	)
	if err != nil {
		return Purchase{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Purchase{}, err
	}

	selectQuery := `SELECT ` + selectColumns + ` FROM purchases WHERE id = ?`

	return scanPurchase(r.db.QueryRowContext(ctx, selectQuery, id))
}

func (r *Repository) Update(ctx context.Context, id uint64, input CreatePurchaseInput) (Purchase, error) {
	purchaseDate, err := time.Parse("2006-01-02", input.PurchaseDate)
	if err != nil {
		return Purchase{}, err
	}

	operationDate, err := nullableDate(input.OperationDate)
	if err != nil {
		return Purchase{}, err
	}

	valueDate, err := nullableDate(input.ValueDate)
	if err != nil {
		return Purchase{}, err
	}

	updateQuery := `
		UPDATE purchases
		SET
			merchant = ?,
			payment_method = ?,
			category_id = ?,
			account_id = ?,
			amount = ?,
			purchase_date = ?,
			note = ?,
			reference = ?,
			operation_label = ?,
			additional_info = ?,
			sub_category = ?,
			operation_date = ?,
			value_date = ?,
			is_reconciled = ?,
			statement_reference = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.db.ExecContext(
		ctx,
		updateQuery,
		input.Merchant,
		input.PaymentMethod,
		input.CategoryID,
		input.AccountID,
		input.Amount,
		purchaseDate,
		input.Note,
		input.Reference,
		input.OperationLabel,
		input.AdditionalInfo,
		input.SubCategory,
		operationDate,
		valueDate,
		input.IsReconciled,
		input.StatementReference,
		id,
	)
	if err != nil {
		return Purchase{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return Purchase{}, err
	}

	if rowsAffected == 0 {
		return Purchase{}, sql.ErrNoRows
	}

	selectQuery := `SELECT ` + selectColumns + ` FROM purchases WHERE id = ?`

	return scanPurchase(r.db.QueryRowContext(ctx, selectQuery, id))
}

func (r *Repository) Delete(ctx context.Context, id uint64) error {
	deleteQuery := `
		DELETE FROM purchases
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, deleteQuery, id)
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
