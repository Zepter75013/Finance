package auth

import "time"

type Session struct {
	Token     string    `json:"-"`
	UserID    uint64    `json:"-"`
	ExpiresAt time.Time `json:"-"`
	CreatedAt time.Time `json:"-"`
}
