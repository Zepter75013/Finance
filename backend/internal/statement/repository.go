package statement

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ErrStatementLocked is returned when trying to overwrite a locked statement —
// the caller must unlock it first, preventing accidental edits.
var ErrStatementLocked = errors.New("statement is locked")

const selectColumns = `
	id,
	account_id,
	reference,
	statement_date,
	period_start,
	period_end,
	start_balance,
	end_balance,
	is_locked,
	created_at,
	updated_at
`

func scanStatement(scanner interface {
	Scan(dest ...any) error
}) (BankStatement, error) {
	var s BankStatement
	var statementDate sql.NullTime
	var periodStart sql.NullTime
	var periodEnd sql.NullTime

	err := scanner.Scan(
		&s.ID,
		&s.AccountID,
		&s.Reference,
		&statementDate,
		&periodStart,
		&periodEnd,
		&s.StartBalance,
		&s.EndBalance,
		&s.IsLocked,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		return BankStatement{}, err
	}

	if statementDate.Valid {
		formatted := statementDate.Time.Format("2006-01-02")
		s.StatementDate = &formatted
	}

	if periodStart.Valid {
		formatted := periodStart.Time.Format("2006-01-02")
		s.PeriodStart = &formatted
	}

	if periodEnd.Valid {
		formatted := periodEnd.Time.Format("2006-01-02")
		s.PeriodEnd = &formatted
	}

	return s, nil
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

func (r *Repository) List(ctx context.Context, accountID uint64) ([]BankStatement, error) {
	query := `
		SELECT ` + selectColumns + `
		FROM bank_statements
		WHERE account_id = ?
		ORDER BY statement_date DESC, id DESC
	`

	rows, err := r.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statements := make([]BankStatement, 0)

	for rows.Next() {
		s, err := scanStatement(rows)
		if err != nil {
			return nil, err
		}

		statements = append(statements, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return statements, nil
}

func (r *Repository) FindByID(ctx context.Context, id uint64) (BankStatement, error) {
	query := `SELECT ` + selectColumns + ` FROM bank_statements WHERE id = ?`

	return scanStatement(r.db.QueryRowContext(ctx, query, id))
}

func (r *Repository) FindByReference(ctx context.Context, accountID uint64, reference string) (BankStatement, error) {
	query := `SELECT ` + selectColumns + ` FROM bank_statements WHERE account_id = ? AND reference = ?`

	return scanStatement(r.db.QueryRowContext(ctx, query, accountID, reference))
}

// Upsert creates the statement identified by its (unique) reference, or
// updates it in place if it already exists — the reconciliation screen just
// "saves" the current relevé, whether it's new or being revisited. It refuses
// to touch a statement that has been locked, as a server-side safety net on
// top of the frontend's own guard.
func (r *Repository) Upsert(ctx context.Context, input UpsertStatementInput) (BankStatement, error) {
	existing, err := r.FindByReference(ctx, input.AccountID, input.Reference)
	if err == nil && existing.IsLocked {
		return BankStatement{}, ErrStatementLocked
	} else if err != nil && err != sql.ErrNoRows {
		return BankStatement{}, err
	}

	statementDate, err := nullableDate(input.StatementDate)
	if err != nil {
		return BankStatement{}, err
	}

	periodStart, err := nullableDate(input.PeriodStart)
	if err != nil {
		return BankStatement{}, err
	}

	periodEnd, err := nullableDate(input.PeriodEnd)
	if err != nil {
		return BankStatement{}, err
	}

	query := `
		INSERT INTO bank_statements (
			account_id,
			reference,
			statement_date,
			period_start,
			period_end,
			start_balance,
			end_balance
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			statement_date = VALUES(statement_date),
			period_start = VALUES(period_start),
			period_end = VALUES(period_end),
			start_balance = VALUES(start_balance),
			end_balance = VALUES(end_balance)
	`

	_, err = r.db.ExecContext(
		ctx,
		query,
		input.AccountID,
		input.Reference,
		statementDate,
		periodStart,
		periodEnd,
		input.StartBalance,
		input.EndBalance,
	)
	if err != nil {
		return BankStatement{}, err
	}

	return r.FindByReference(ctx, input.AccountID, input.Reference)
}

func (r *Repository) SetLocked(ctx context.Context, id uint64, locked bool) (BankStatement, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE bank_statements SET is_locked = ? WHERE id = ?`, locked, id)
	if err != nil {
		return BankStatement{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return BankStatement{}, err
	}

	if rowsAffected == 0 {
		if _, err := r.FindByID(ctx, id); err != nil {
			return BankStatement{}, sql.ErrNoRows
		}
	}

	return r.FindByID(ctx, id)
}

const pdfSelectColumns = `
	id,
	statement_id,
	original_filename,
	created_at
`

func scanStatementPdf(scanner interface {
	Scan(dest ...any) error
}) (StatementPdf, error) {
	var p StatementPdf

	err := scanner.Scan(&p.ID, &p.StatementID, &p.OriginalFilename, &p.CreatedAt)
	if err != nil {
		return StatementPdf{}, err
	}

	return p, nil
}

// ListPdfs returns every PDF attached to a statement, oldest first —
// attaching several PDFs is allowed regardless of lock state, since storing
// the scanned relevé isn't considered "modifying" the reconciliation data.
func (r *Repository) ListPdfs(ctx context.Context, statementID uint64) ([]StatementPdf, error) {
	query := `SELECT ` + pdfSelectColumns + ` FROM bank_statement_pdfs WHERE statement_id = ? ORDER BY created_at ASC, id ASC`

	rows, err := r.db.QueryContext(ctx, query, statementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pdfs := make([]StatementPdf, 0)

	for rows.Next() {
		p, err := scanStatementPdf(rows)
		if err != nil {
			return nil, err
		}

		pdfs = append(pdfs, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pdfs, nil
}

// AddPdf records a newly uploaded PDF file. filename is the name it's stored
// under on disk (internal, never exposed); originalFilename is what the user
// picked, kept for display/download purposes.
func (r *Repository) AddPdf(ctx context.Context, statementID uint64, filename, originalFilename string) (StatementPdf, error) {
	result, err := r.db.ExecContext(
		ctx,
		`INSERT INTO bank_statement_pdfs (statement_id, filename, original_filename) VALUES (?, ?, ?)`,
		statementID,
		filename,
		originalFilename,
	)
	if err != nil {
		return StatementPdf{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return StatementPdf{}, err
	}

	return r.FindPdf(ctx, uint64(id))
}

func (r *Repository) FindPdf(ctx context.Context, pdfID uint64) (StatementPdf, error) {
	query := `SELECT ` + pdfSelectColumns + ` FROM bank_statement_pdfs WHERE id = ?`

	return scanStatementPdf(r.db.QueryRowContext(ctx, query, pdfID))
}

// PdfDiskFilename returns the internal on-disk filename for a PDF row —
// separate from FindPdf since callers outside this file never need it.
func (r *Repository) PdfDiskFilename(ctx context.Context, pdfID uint64) (string, error) {
	var filename string
	err := r.db.QueryRowContext(ctx, `SELECT filename FROM bank_statement_pdfs WHERE id = ?`, pdfID).Scan(&filename)
	return filename, err
}

func (r *Repository) DeletePdf(ctx context.Context, pdfID uint64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM bank_statement_pdfs WHERE id = ?`, pdfID)
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

func (r *Repository) Delete(ctx context.Context, id uint64) error {
	existing, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.IsLocked {
		return ErrStatementLocked
	}

	result, err := r.db.ExecContext(ctx, `DELETE FROM bank_statements WHERE id = ?`, id)
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
