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
	// IsAdmin autorise la gestion des utilisateurs (créer/modifier/supprimer
	// un utilisateur, assigner des comptes) — voir user.Handler.
	IsAdmin   bool      `json:"is_admin"`
	// EmailAlertsEnabled contrôle la réception du résumé quotidien par email
	// (budgets dépassés, échéances de récurrence) — préférence personnelle,
	// sans lien avec IsAdmin.
	EmailAlertsEnabled bool      `json:"email_alerts_enabled"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
