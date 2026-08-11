package user

import "time"

type User struct {
	ID           uint64    `json:"id"`
	Username     string    `json:"username"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	AvatarURL    *string   `json:"avatar_url"`
	PasswordHash string    `json:"-"`
	// AccountIDs liste les comptes explicitement assignés à cet utilisateur.
	// Une liste vide (mais non nulle) signifie "restreint à aucun compte" ;
	// une liste nulle/absente signifie "jamais configuré", donc accès à tous
	// les comptes — voir account.Repository.UserCanAccess.
	AccountIDs []uint64  `json:"account_ids"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
