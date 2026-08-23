package budget

import "time"

type CategoryBudget struct {
	ID         uint64    `json:"id"`
	CategoryID uint64    `json:"category_id"`
	MonthKey   string    `json:"month_key"`
	Amount     float64   `json:"amount"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
