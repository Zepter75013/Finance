package account

type Account struct {
	ID            uint64  `json:"id"`
	Name          string  `json:"name"`
	PurchaseCount uint64  `json:"purchase_count"`
	IncomeCount   uint64  `json:"income_count"`
	TotalExpense  float64 `json:"total_expense"`
	TotalIncome   float64 `json:"total_income"`
	CategoryCount uint64  `json:"category_count"`
}
