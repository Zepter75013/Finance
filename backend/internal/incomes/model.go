package incomes

import "time"

type Income struct {
	ID                 uint64    `json:"id"`
	AccountID          uint64    `json:"account_id"`
	Source             string    `json:"source"`
	Amount             float64   `json:"amount"`
	IncomeDate         time.Time `json:"income_date"`
	Note               string    `json:"note"`
	Reference          string    `json:"reference"`
	OperationLabel     string    `json:"operation_label"`
	AdditionalInfo     string    `json:"additional_info"`
	OperationType      string    `json:"operation_type"`
	Category           string    `json:"category"`
	SubCategory        string    `json:"sub_category"`
	OperationDate      *string   `json:"operation_date"`
	ValueDate          *string   `json:"value_date"`
	IsReconciled       bool      `json:"is_reconciled"`
	StatementReference string    `json:"statement_reference"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type CreateIncomeInput struct {
	AccountID          uint64  `json:"account_id"`
	Source             string  `json:"source"`
	Amount             float64 `json:"amount"`
	IncomeDate         string  `json:"income_date"`
	Note               string  `json:"note"`
	Reference          string  `json:"reference"`
	OperationLabel     string  `json:"operation_label"`
	AdditionalInfo     string  `json:"additional_info"`
	OperationType      string  `json:"operation_type"`
	Category           string  `json:"category"`
	SubCategory        string  `json:"sub_category"`
	OperationDate      string  `json:"operation_date"`
	ValueDate          string  `json:"value_date"`
	IsReconciled       bool    `json:"is_reconciled"`
	StatementReference string  `json:"statement_reference"`
}
