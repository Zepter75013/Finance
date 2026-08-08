package subcategory

import "time"

type SubCategory struct {
	ID         uint64    `json:"id"`
	CategoryID uint64    `json:"category_id"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
