package recurring

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
	type,
	merchant,
	source,
	category_id,
	payment_method,
	operation_type,
	category,
	sub_category,
	amount,
	day_of_month,
	note,
	is_active,
	next_run_date,
	created_at,
	updated_at
`

func scanRecurring(scanner interface {
	Scan(dest ...any) error
}) (RecurringTransaction, error) {
	var rt RecurringTransaction
	var merchant sql.NullString
	var source sql.NullString
	var categoryID sql.NullInt64
	var paymentMethod sql.NullString
	var operationType sql.NullString
	var category sql.NullString
	var note sql.NullString

	err := scanner.Scan(
		&rt.ID,
		&rt.AccountID,
		&rt.Type,
		&merchant,
		&source,
		&categoryID,
		&paymentMethod,
		&operationType,
		&category,
		&rt.SubCategory,
		&rt.Amount,
		&rt.DayOfMonth,
		&note,
		&rt.IsActive,
		&rt.NextRunDate,
		&rt.CreatedAt,
		&rt.UpdatedAt,
	)
	if err != nil {
		return RecurringTransaction{}, err
	}

	if merchant.Valid {
		rt.Merchant = &merchant.String
	}
	if source.Valid {
		rt.Source = &source.String
	}
	if categoryID.Valid {
		id := uint64(categoryID.Int64)
		rt.CategoryID = &id
	}
	if paymentMethod.Valid {
		rt.PaymentMethod = &paymentMethod.String
	}
	if operationType.Valid {
		rt.OperationType = &operationType.String
	}
	if category.Valid {
		rt.Category = &category.String
	}
	if note.Valid {
		rt.Note = &note.String
	}

	return rt, nil
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func nullableCategoryID(value uint64) *uint64 {
	if value == 0 {
		return nil
	}
	return &value
}

func (r *Repository) ListByAccount(ctx context.Context, accountID uint64) ([]RecurringTransaction, error) {
	query := `
		SELECT ` + selectColumns + `
		FROM recurring_transactions
		WHERE account_id = ?
		ORDER BY day_of_month ASC, id ASC
	`

	rows, err := r.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]RecurringTransaction, 0)

	for rows.Next() {
		item, err := scanRecurring(rows)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

// DueForExecution renvoie les gabarits actifs dont la prochaine occurrence
// est arrivée ou dépassée — utilisé par le planificateur de fond.
func (r *Repository) DueForExecution(ctx context.Context, asOf time.Time) ([]RecurringTransaction, error) {
	query := `
		SELECT ` + selectColumns + `
		FROM recurring_transactions
		WHERE is_active = 1 AND next_run_date <= ?
		ORDER BY next_run_date ASC
	`

	rows, err := r.db.QueryContext(ctx, query, asOf.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]RecurringTransaction, 0)

	for rows.Next() {
		item, err := scanRecurring(rows)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *Repository) GetByID(ctx context.Context, id uint64) (RecurringTransaction, error) {
	query := `SELECT ` + selectColumns + ` FROM recurring_transactions WHERE id = ?`

	return scanRecurring(r.db.QueryRowContext(ctx, query, id))
}

func (r *Repository) Create(ctx context.Context, input CreateRecurringInput) (RecurringTransaction, error) {
	nextRun, err := time.Parse("2006-01-02", input.FirstRunDate)
	if err != nil {
		return RecurringTransaction{}, err
	}

	dayOfMonth := nextRun.Day()

	query := `
		INSERT INTO recurring_transactions (
			account_id, type, merchant, source, category_id, payment_method,
			operation_type, category, sub_category, amount, day_of_month, note,
			is_active, next_run_date
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		input.AccountID,
		input.Type,
		nullableString(input.Merchant),
		nullableString(input.Source),
		nullableCategoryID(input.CategoryID),
		nullableString(input.PaymentMethod),
		nullableString(input.OperationType),
		nullableString(input.Category),
		input.SubCategory,
		input.Amount,
		dayOfMonth,
		nullableString(input.Note),
		input.IsActive,
		nextRun.Format("2006-01-02"),
	)
	if err != nil {
		return RecurringTransaction{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return RecurringTransaction{}, err
	}

	return r.GetByID(ctx, uint64(id))
}

func (r *Repository) Update(ctx context.Context, id uint64, input CreateRecurringInput) (RecurringTransaction, error) {
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return RecurringTransaction{}, err
	}

	nextRun := existing.NextRunDate
	dayOfMonth := existing.DayOfMonth

	// Une date de première occurrence différente de la prochaine échéance
	// actuelle signifie que l'utilisateur a explicitement décalé la
	// récurrence — on adopte cette nouvelle date (et son jour du mois pour
	// les occurrences suivantes) telle quelle.
	if input.FirstRunDate != existing.NextRunDate.Format("2006-01-02") {
		parsed, err := time.Parse("2006-01-02", input.FirstRunDate)
		if err != nil {
			return RecurringTransaction{}, err
		}

		nextRun = parsed
		dayOfMonth = parsed.Day()
	}

	query := `
		UPDATE recurring_transactions
		SET
			type = ?,
			merchant = ?,
			source = ?,
			category_id = ?,
			payment_method = ?,
			operation_type = ?,
			category = ?,
			sub_category = ?,
			amount = ?,
			day_of_month = ?,
			note = ?,
			is_active = ?,
			next_run_date = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err = r.db.ExecContext(ctx, query,
		input.Type,
		nullableString(input.Merchant),
		nullableString(input.Source),
		nullableCategoryID(input.CategoryID),
		nullableString(input.PaymentMethod),
		nullableString(input.OperationType),
		nullableString(input.Category),
		input.SubCategory,
		input.Amount,
		dayOfMonth,
		nullableString(input.Note),
		input.IsActive,
		nextRun.Format("2006-01-02"),
		id,
	)
	if err != nil {
		return RecurringTransaction{}, err
	}

	return r.GetByID(ctx, id)
}

func (r *Repository) Delete(ctx context.Context, id uint64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM recurring_transactions WHERE id = ?`, id)
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

// AdvanceAfterExecution avance la prochaine occurrence d'un gabarit après
// que la transaction du jour a été créée avec succès.
func (r *Repository) AdvanceAfterExecution(ctx context.Context, id uint64, scheduledDate time.Time, dayOfMonth int) error {
	next := advanceOneMonth(scheduledDate, dayOfMonth)

	_, err := r.db.ExecContext(ctx,
		`UPDATE recurring_transactions SET next_run_date = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		next.Format("2006-01-02"), id,
	)

	return err
}
