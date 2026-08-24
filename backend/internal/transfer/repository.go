package transfer

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
	from_account_id,
	to_account_id,
	amount,
	transfer_date,
	note,
	from_is_reconciled,
	from_statement_reference,
	to_is_reconciled,
	to_statement_reference,
	origin_type,
	origin_payload,
	created_at,
	updated_at
`

func scanTransfer(scanner interface {
	Scan(dest ...any) error
}) (Transfer, error) {
	var t Transfer
	var originType, originPayload sql.NullString

	err := scanner.Scan(
		&t.ID,
		&t.FromAccountID,
		&t.ToAccountID,
		&t.Amount,
		&t.TransferDate,
		&t.Note,
		&t.FromIsReconciled,
		&t.FromStatementReference,
		&t.ToIsReconciled,
		&t.ToStatementReference,
		&originType,
		&originPayload,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		return Transfer{}, err
	}

	if originType.Valid {
		t.OriginType = &originType.String
	}
	if originPayload.Valid {
		t.OriginPayload = &originPayload.String
	}

	return t, nil
}

// ListByAccount renvoie les virements touchant accountID, qu'il en soit la
// source ou la destination — les deux directions sont nécessaires pour le
// solde réel et l'écran Pointage, chacun scopé à un seul compte à la fois.
func (r *Repository) ListByAccount(ctx context.Context, accountID uint64) ([]Transfer, error) {
	query := `
		SELECT ` + selectColumns + `
		FROM transfers
		WHERE from_account_id = ? OR to_account_id = ?
		ORDER BY transfer_date DESC, id DESC
	`

	rows, err := r.db.QueryContext(ctx, query, accountID, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transfers := make([]Transfer, 0)

	for rows.Next() {
		t, err := scanTransfer(rows)
		if err != nil {
			return nil, err
		}

		transfers = append(transfers, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return transfers, nil
}

func (r *Repository) Create(ctx context.Context, input TransferInput) (Transfer, error) {
	transferDate, err := time.Parse("2006-01-02", input.TransferDate)
	if err != nil {
		return Transfer{}, err
	}

	insertQuery := `
		INSERT INTO transfers (
			from_account_id, to_account_id, amount, transfer_date, note,
			from_is_reconciled, from_statement_reference, to_is_reconciled, to_statement_reference,
			origin_type, origin_payload
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, insertQuery,
		input.FromAccountID, input.ToAccountID, input.Amount, transferDate, input.Note,
		input.FromIsReconciled, input.FromStatementReference, input.ToIsReconciled, input.ToStatementReference,
		input.OriginType, input.OriginPayload,
	)
	if err != nil {
		return Transfer{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Transfer{}, err
	}

	return r.FindByID(ctx, uint64(id))
}

func (r *Repository) Update(ctx context.Context, id uint64, input TransferInput) (Transfer, error) {
	transferDate, err := time.Parse("2006-01-02", input.TransferDate)
	if err != nil {
		return Transfer{}, err
	}

	updateQuery := `
		UPDATE transfers
		SET
			from_account_id = ?, to_account_id = ?, amount = ?, transfer_date = ?, note = ?,
			from_is_reconciled = ?, from_statement_reference = ?, to_is_reconciled = ?, to_statement_reference = ?,
			origin_type = ?, origin_payload = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, updateQuery,
		input.FromAccountID, input.ToAccountID, input.Amount, transferDate, input.Note,
		input.FromIsReconciled, input.FromStatementReference, input.ToIsReconciled, input.ToStatementReference,
		input.OriginType, input.OriginPayload,
		id,
	)
	if err != nil {
		return Transfer{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return Transfer{}, err
	}

	if rowsAffected == 0 {
		return Transfer{}, sql.ErrNoRows
	}

	return r.FindByID(ctx, id)
}

func (r *Repository) FindByID(ctx context.Context, id uint64) (Transfer, error) {
	query := `SELECT ` + selectColumns + ` FROM transfers WHERE id = ?`

	return scanTransfer(r.db.QueryRowContext(ctx, query, id))
}

func (r *Repository) Delete(ctx context.Context, id uint64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM transfers WHERE id = ?`, id)
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
