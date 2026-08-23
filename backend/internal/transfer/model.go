package transfer

import "time"

// Transfer débite FromAccountID et crédite ToAccountID. Deux paires de
// champs de pointage indépendantes (From*/To*) car chaque compte a son
// propre historique de relevés bancaires, sans lien entre eux — un
// virement peut être pointé côté source avant de l'être côté destination
// (ou jamais, si le compte destination n'a pas de relevé — ex: un Livret).
type Transfer struct {
	ID                     uint64    `json:"id"`
	FromAccountID          uint64    `json:"from_account_id"`
	ToAccountID            uint64    `json:"to_account_id"`
	Amount                 float64   `json:"amount"`
	TransferDate           time.Time `json:"transfer_date"`
	Note                   string    `json:"note"`
	FromIsReconciled       bool      `json:"from_is_reconciled"`
	FromStatementReference string    `json:"from_statement_reference"`
	ToIsReconciled         bool      `json:"to_is_reconciled"`
	ToStatementReference   string    `json:"to_statement_reference"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type TransferInput struct {
	FromAccountID          uint64  `json:"from_account_id"`
	ToAccountID            uint64  `json:"to_account_id"`
	Amount                 float64 `json:"amount"`
	TransferDate           string  `json:"transfer_date"`
	Note                   string  `json:"note"`
	FromIsReconciled       bool    `json:"from_is_reconciled"`
	FromStatementReference string  `json:"from_statement_reference"`
	ToIsReconciled         bool    `json:"to_is_reconciled"`
	ToStatementReference   string  `json:"to_statement_reference"`
}
