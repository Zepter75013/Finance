package recurring

import "time"

const (
	TypeAchat  = "achat"
	TypeRevenu = "revenu"
)

type RecurringTransaction struct {
	ID            uint64    `json:"id"`
	AccountID     uint64    `json:"account_id"`
	Type          string    `json:"type"`
	Merchant      *string   `json:"merchant"`
	Source        *string   `json:"source"`
	CategoryID    *uint64   `json:"category_id"`
	PaymentMethod *string   `json:"payment_method"`
	OperationType *string   `json:"operation_type"`
	Category      *string   `json:"category"`
	SubCategory   string    `json:"sub_category"`
	Amount        float64   `json:"amount"`
	DayOfMonth    int       `json:"day_of_month"`
	Note          *string   `json:"note"`
	IsActive      bool      `json:"is_active"`
	NextRunDate   time.Time `json:"next_run_date"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateRecurringInput struct {
	AccountID     uint64  `json:"account_id"`
	Type          string  `json:"type"`
	Merchant      string  `json:"merchant"`
	Source        string  `json:"source"`
	CategoryID    uint64  `json:"category_id"`
	PaymentMethod string  `json:"payment_method"`
	OperationType string  `json:"operation_type"`
	Category      string  `json:"category"`
	SubCategory   string  `json:"sub_category"`
	Amount        float64 `json:"amount"`
	// FirstRunDate ("2006-01-02") fixe explicitement la date de la première
	// occurrence — son jour du mois devient aussi la règle de récurrence
	// pour les occurrences suivantes (pas besoin d'un champ séparé qui
	// pourrait la contredire).
	FirstRunDate string `json:"first_run_date"`
	Note         string `json:"note"`
	IsActive     bool   `json:"is_active"`
}
