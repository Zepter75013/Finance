package account

type Account struct {
	ID            uint64  `json:"id"`
	Name          string  `json:"name"`
	PurchaseCount uint64  `json:"purchase_count"`
	IncomeCount   uint64  `json:"income_count"`
	TotalExpense  float64 `json:"total_expense"`
	TotalIncome   float64 `json:"total_income"`
	CategoryCount uint64  `json:"category_count"`
	TransferCount uint64  `json:"transfer_count"`
	// OpeningBalanceAmount/Date servent d'ancrage à computeRealBalance quand
	// aucun relevé n'est verrouillé (compte type Livret, qui n'en a
	// structurellement jamais) — nil tant qu'aucun n'est déclaré.
	OpeningBalanceAmount *float64 `json:"opening_balance_amount"`
	OpeningBalanceDate   *string  `json:"opening_balance_date"`
}
