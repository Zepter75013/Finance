package auth

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"finance/backend/internal/mailer"
	"finance/backend/internal/user"

	"golang.org/x/crypto/bcrypt"
)

const CookieName = "session_token"

type Handler struct {
	sessions *Repository
	users    *user.Repository
	mailer   *mailer.Mailer
}

func NewHandler(sessions *Repository, users *user.Repository, mailer *mailer.Mailer) *Handler {
	return &Handler{sessions: sessions, users: users, mailer: mailer}
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, r *http.Request, session Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	payload.Username = strings.TrimSpace(payload.Username)

	if payload.Username == "" || payload.Password == "" {
		http.Error(w, "identifiants invalides", http.StatusUnauthorized)
		return
	}

	account, err := h.users.FindByUsername(r.Context(), payload.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "identifiants invalides", http.StatusUnauthorized)
			return
		}

		http.Error(w, "échec de connexion", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(payload.Password)); err != nil {
		http.Error(w, "identifiants invalides", http.StatusUnauthorized)
		return
	}

	session, err := h.sessions.CreateSession(r.Context(), account.ID)
	if err != nil {
		http.Error(w, "échec de connexion", http.StatusInternalServerError)
		return
	}

	h.setSessionCookie(w, r, session)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(accountSummary(account))
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if cookie, err := r.Cookie(CookieName); err == nil {
		_ = h.sessions.DeleteSession(r.Context(), cookie.Value)
	}

	h.clearSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) currentUser(r *http.Request) (user.User, error) {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return user.User{}, err
	}

	session, err := h.sessions.FindValidSession(r.Context(), cookie.Value)
	if err != nil {
		return user.User{}, err
	}

	return h.users.FindByID(r.Context(), session.UserID)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	account, err := h.currentUser(r)
	if err != nil {
		http.Error(w, "non authentifié", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(accountSummary(account))
}

// accountSummary returns the subset of a user's profile that the frontend
// needs for its own session (sidebar avatar, greeting) — never the password
// hash.
func accountSummary(account user.User) map[string]any {
	return map[string]any{
		"id":         account.ID,
		"username":   account.Username,
		"first_name": account.FirstName,
		"last_name":  account.LastName,
		"avatar_url": account.AvatarURL,
	}
}

func (h *Handler) applyPasswordChange(w http.ResponseWriter, r *http.Request, account user.User, newPassword string) {
	if len(newPassword) < 4 {
		http.Error(w, "le nouveau mot de passe doit contenir au moins 4 caractères", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "échec de la mise à jour", http.StatusInternalServerError)
		return
	}

	if err := h.users.UpdatePassword(r.Context(), account.ID, string(hash)); err != nil {
		http.Error(w, "échec de la mise à jour", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ChangePassword updates the password of the currently authenticated user
// (used from within the app, e.g. the sidebar's "modifier le mot de passe" action).
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	account, err := h.currentUser(r)
	if err != nil {
		http.Error(w, "non authentifié", http.StatusUnauthorized)
		return
	}

	var payload struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(payload.CurrentPassword)); err != nil {
		http.Error(w, "mot de passe actuel incorrect", http.StatusUnauthorized)
		return
	}

	h.applyPasswordChange(w, r, account, payload.NewPassword)
}

// RequestResetCode sends a one-time 6-digit code to the account's email
// address (its username) so the user can reset their password from the login
// screen without knowing their current one. The response never reveals
// whether the username actually exists, to avoid leaking account presence.
func (h *Handler) RequestResetCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	payload.Username = strings.TrimSpace(payload.Username)

	if payload.Username == "" {
		http.Error(w, "identifiant requis", http.StatusBadRequest)
		return
	}

	account, err := h.users.FindByUsername(r.Context(), payload.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		http.Error(w, "échec de l'envoi du code", http.StatusInternalServerError)
		return
	}

	code, err := h.sessions.CreateResetCode(r.Context(), account.ID)
	if err != nil {
		http.Error(w, "échec de l'envoi du code", http.StatusInternalServerError)
		return
	}

	greeting := "Bonjour"
	if account.FirstName != "" {
		greeting = "Bonjour " + account.FirstName
	}

	body := greeting + ",\n\n" +
		"Voici ton code de réinitialisation de mot de passe : " + code + "\n" +
		"Ce code est valable 10 minutes.\n\n" +
		"Si tu n'es pas à l'origine de cette demande, ignore cet email."

	if h.mailer != nil && h.mailer.Configured() {
		if err := h.mailer.Send(account.Username, "Code de réinitialisation - Finance UI", body); err != nil {
			log.Printf("échec de l'envoi de l'email, code de secours pour %s : %s (erreur : %v)", account.Username, code, err)
		}
	} else {
		log.Printf("SMTP non configuré — code de réinitialisation pour %s : %s", account.Username, code)
	}

	w.WriteHeader(http.StatusNoContent)
}

// ResetPasswordWithCode verifies the one-time code sent by email and, if
// valid, updates the password. No prior password is required.
func (h *Handler) ResetPasswordWithCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Username    string `json:"username"`
		Code        string `json:"code"`
		NewPassword string `json:"newPassword"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	payload.Username = strings.TrimSpace(payload.Username)
	payload.Code = strings.TrimSpace(payload.Code)

	if payload.Username == "" || payload.Code == "" {
		http.Error(w, "identifiant et code requis", http.StatusBadRequest)
		return
	}

	account, err := h.users.FindByUsername(r.Context(), payload.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "code invalide ou expiré", http.StatusUnauthorized)
			return
		}

		http.Error(w, "échec de la mise à jour", http.StatusInternalServerError)
		return
	}

	valid, err := h.sessions.ConsumeResetCode(r.Context(), account.ID, payload.Code)
	if err != nil {
		http.Error(w, "échec de la mise à jour", http.StatusInternalServerError)
		return
	}

	if !valid {
		http.Error(w, "code invalide ou expiré", http.StatusUnauthorized)
		return
	}

	h.applyPasswordChange(w, r, account, payload.NewPassword)
}
