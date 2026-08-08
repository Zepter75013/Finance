package purchase

import "time"

type Purchase struct {
	ID                 uint64    `json:"id"`
	Merchant           string    `json:"merchant"`
	PaymentMethod      string    `json:"payment_method"`
	CategoryID         uint64    `json:"category_id"`
	AccountID          uint64    `json:"account_id"`
	Amount             float64   `json:"amount"`
	PurchaseDate       time.Time `json:"purchase_date"`
	Note               string    `json:"note"`
	Reference          string    `json:"reference"`
	OperationLabel     string    `json:"operation_label"`
	AdditionalInfo     string    `json:"additional_info"`
	SubCategory        string    `json:"sub_category"`
	OperationDate      *string   `json:"operation_date"`
	ValueDate          *string   `json:"value_date"`
	IsReconciled       bool      `json:"is_reconciled"`
	StatementReference string    `json:"statement_reference"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type CreatePurchaseInput struct {
	Merchant           string  `json:"merchant"`
	PaymentMethod      string  `json:"payment_method"`
	CategoryID         uint64  `json:"category_id"`
	AccountID          uint64  `json:"account_id"`
	Amount             float64 `json:"amount"`
	PurchaseDate       string  `json:"purchase_date"`
	Note               string  `json:"note"`
	Reference          string  `json:"reference"`
	OperationLabel     string  `json:"operation_label"`
	AdditionalInfo     string  `json:"additional_info"`
	SubCategory        string  `json:"sub_category"`
	OperationDate      string  `json:"operation_date"`
	ValueDate          string  `json:"value_date"`
	IsReconciled       bool    `json:"is_reconciled"`
	StatementReference string  `json:"statement_reference"`
}
