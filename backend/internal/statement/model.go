package statement

import "time"

type BankStatement struct {
	ID            uint64    `json:"id"`
	AccountID     uint64    `json:"account_id"`
	Reference     string    `json:"reference"`
	StatementDate *string   `json:"statement_date"`
	PeriodStart   *string   `json:"period_start"`
	PeriodEnd     *string   `json:"period_end"`
	StartBalance  float64   `json:"start_balance"`
	EndBalance    float64   `json:"end_balance"`
	IsLocked      bool      `json:"is_locked"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type StatementPdf struct {
	ID               uint64    `json:"id"`
	StatementID      uint64    `json:"statement_id"`
	OriginalFilename string    `json:"original_filename"`
	CreatedAt        time.Time `json:"created_at"`
}

type UpsertStatementInput struct {
	AccountID     uint64  `json:"account_id"`
	Reference     string  `json:"reference"`
	StatementDate string  `json:"statement_date"`
	PeriodStart   string  `json:"period_start"`
	PeriodEnd     string  `json:"period_end"`
	StartBalance  float64 `json:"start_balance"`
	EndBalance    float64 `json:"end_balance"`
}
