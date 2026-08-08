package incomes

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
	account_id,
	source,
	amount,
	income_date,
	note,
	reference,
	operation_label,
	additional_info,
	operation_type,
	category,
	sub_category,
	operation_date,
	value_date,
	is_reconciled,
	statement_reference,
	created_at,
	updated_at
`

func scanIncome(scanner interface {
	Scan(dest ...any) error
}) (Income, error) {
	var income Income
	var note sql.NullString
	var additionalInfo sql.NullString
	var operationDate sql.NullTime
	var valueDate sql.NullTime

	err := scanner.Scan(
		&income.ID,
		&income.AccountID,
		&income.Source,
		&income.Amount,
		&income.IncomeDate,
		&note,
		&income.Reference,
		&income.OperationLabel,
		&additionalInfo,
		&income.OperationType,
		&income.Category,
		&income.SubCategory,
		&operationDate,
		&valueDate,
		&income.IsReconciled,
		&income.StatementReference,
		&income.CreatedAt,
		&income.UpdatedAt,
	)
	if err != nil {
		return Income{}, err
	}

	if note.Valid {
		income.Note = note.String
	}

	if additionalInfo.Valid {
		income.AdditionalInfo = additionalInfo.String
	}

	if operationDate.Valid {
		formatted := operationDate.Time.Format("2006-01-02")
		income.OperationDate = &formatted
	}

	if valueDate.Valid {
		formatted := valueDate.Time.Format("2006-01-02")
		income.ValueDate = &formatted
	}

	return income, nil
}

func (r *Repository) List(ctx context.Context, accountID uint64) ([]Income, error) {
	query := `SELECT ` + selectColumns + ` FROM incomes WHERE account_id = ? ORDER BY income_date DESC, id DESC`

	rows, err := r.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	incomes := make([]Income, 0)

	for rows.Next() {
		income, err := scanIncome(rows)
		if err != nil {
			return nil, err
		}

		incomes = append(incomes, income)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return incomes, nil
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

func (r *Repository) Create(ctx context.Context, input CreateIncomeInput) (Income, error) {
	incomeDate, err := time.Parse("2006-01-02", input.IncomeDate)
	if err != nil {
		return Income{}, err
	}

	operationDate, err := nullableDate(input.OperationDate)
	if err != nil {
		return Income{}, err
	}

	valueDate, err := nullableDate(input.ValueDate)
	if err != nil {
		return Income{}, err
	}

	insertQuery := `
		INSERT INTO incomes (
			account_id,
			source,
			amount,
			income_date,
			note,
			reference,
			operation_label,
			additional_info,
			operation_type,
			category,
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
		input.AccountID,
		input.Source,
		input.Amount,
		incomeDate,
		input.Note,
		input.Reference,
		input.OperationLabel,
		input.AdditionalInfo,
		input.OperationType,
		input.Category,
		input.SubCategory,
		operationDate,
		valueDate,
		input.IsReconciled,
		input.StatementReference,
	)
	if err != nil {
		return Income{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Income{}, err
	}

	selectQuery := `SELECT ` + selectColumns + ` FROM incomes WHERE id = ?`

	return scanIncome(r.db.QueryRowContext(ctx, selectQuery, id))
}

func (r *Repository) Update(ctx context.Context, id uint64, input CreateIncomeInput) (Income, error) {
	incomeDate, err := time.Parse("2006-01-02", input.IncomeDate)
	if err != nil {
		return Income{}, err
	}

	operationDate, err := nullableDate(input.OperationDate)
	if err != nil {
		return Income{}, err
	}

	valueDate, err := nullableDate(input.ValueDate)
	if err != nil {
		return Income{}, err
	}

	updateQuery := `
		UPDATE incomes
		SET
			account_id = ?,
			source = ?,
			amount = ?,
			income_date = ?,
			note = ?,
			reference = ?,
			operation_label = ?,
			additional_info = ?,
			operation_type = ?,
			category = ?,
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
		input.AccountID,
		input.Source,
		input.Amount,
		incomeDate,
		input.Note,
		input.Reference,
		input.OperationLabel,
		input.AdditionalInfo,
		input.OperationType,
		input.Category,
		input.SubCategory,
		operationDate,
		valueDate,
		input.IsReconciled,
		input.StatementReference,
		id,
	)
	if err != nil {
		return Income{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return Income{}, err
	}

	if rowsAffected == 0 {
		return Income{}, sql.ErrNoRows
	}

	selectQuery := `SELECT ` + selectColumns + ` FROM incomes WHERE id = ?`

	return scanIncome(r.db.QueryRowContext(ctx, selectQuery, id))
}

func (r *Repository) Delete(ctx context.Context, id uint64) error {
	deleteQuery := `
		DELETE FROM incomes
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
