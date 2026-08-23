package auditlog

import "time"

type Entry struct {
	ID         uint64    `json:"id"`
	UserID     *uint64   `json:"user_id"`
	Username   string    `json:"username"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	StatusCode int       `json:"status_code"`
	CreatedAt  time.Time `json:"created_at"`
}
