package user

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	repo *Repository
}

// maxAvatarDataURLLength caps the stored avatar (a base64 data URL produced
// client-side after resizing the image to a small square) to keep rows small.
const maxAvatarDataURLLength = 2_000_000

func normalizeAvatarURL(raw string) (*string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	if len(trimmed) > maxAvatarDataURLLength {
		return nil, errors.New("l'image est trop volumineuse")
	}

	return &trimmed, nil
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	users, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, "failed to load users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(users)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Username   string    `json:"username"`
		FirstName  string    `json:"first_name"`
		LastName   string    `json:"last_name"`
		AvatarURL  string    `json:"avatar_url"`
		Password   string    `json:"password"`
		AccountIDs *[]uint64 `json:"account_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	payload.Username = strings.TrimSpace(payload.Username)
	payload.FirstName = strings.TrimSpace(payload.FirstName)
	payload.LastName = strings.TrimSpace(payload.LastName)

	if payload.Username == "" {
		http.Error(w, "l'identifiant est obligatoire", http.StatusBadRequest)
		return
	}

	if payload.FirstName == "" || payload.LastName == "" {
		http.Error(w, "le prénom et le nom sont obligatoires", http.StatusBadRequest)
		return
	}

	if len(payload.Password) < 4 {
		http.Error(w, "le mot de passe doit contenir au moins 4 caractères", http.StatusBadRequest)
		return
	}

	avatarURL, err := normalizeAvatarURL(payload.AvatarURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := h.repo.FindByUsername(r.Context(), payload.Username); err == nil {
		http.Error(w, "un utilisateur avec cet identifiant existe déjà", http.StatusConflict)
		return
	} else if err != sql.ErrNoRows {
		http.Error(w, "échec de la création", http.StatusInternalServerError)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "échec de la création", http.StatusInternalServerError)
		return
	}

	created, err := h.repo.Create(r.Context(), payload.Username, payload.FirstName, payload.LastName, avatarURL, string(hash), payload.AccountIDs)
	if err != nil {
		http.Error(w, "échec de la création", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(created)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/users/")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var payload struct {
		Username   string    `json:"username"`
		FirstName  string    `json:"first_name"`
		LastName   string    `json:"last_name"`
		AvatarURL  string    `json:"avatar_url"`
		AccountIDs *[]uint64 `json:"account_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	payload.Username = strings.TrimSpace(payload.Username)
	payload.FirstName = strings.TrimSpace(payload.FirstName)
	payload.LastName = strings.TrimSpace(payload.LastName)

	if payload.Username == "" {
		http.Error(w, "l'identifiant est obligatoire", http.StatusBadRequest)
		return
	}

	if payload.FirstName == "" || payload.LastName == "" {
		http.Error(w, "le prénom et le nom sont obligatoires", http.StatusBadRequest)
		return
	}

	avatarURL, err := normalizeAvatarURL(payload.AvatarURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if existing, err := h.repo.FindByUsername(r.Context(), payload.Username); err == nil && existing.ID != id {
		http.Error(w, "un utilisateur avec cet identifiant existe déjà", http.StatusConflict)
		return
	} else if err != nil && err != sql.ErrNoRows {
		http.Error(w, "échec de la mise à jour", http.StatusInternalServerError)
		return
	}

	updated, err := h.repo.Update(r.Context(), id, payload.Username, payload.FirstName, payload.LastName, avatarURL, payload.AccountIDs)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "utilisateur introuvable", http.StatusNotFound)
			return
		}

		http.Error(w, "échec de la mise à jour", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(updated)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/users/")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	count, err := h.repo.Count(r.Context())
	if err != nil {
		http.Error(w, "failed to delete user", http.StatusInternalServerError)
		return
	}

	if count <= 1 {
		http.Error(w, "impossible de supprimer le dernier compte", http.StatusConflict)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "utilisateur introuvable", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to delete user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
