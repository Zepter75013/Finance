package account

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"finance/backend/internal/authctx"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	userID, _ := authctx.UserID(r.Context())

	accounts, err := h.repo.List(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to load accounts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(accounts)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	payload.Name = strings.TrimSpace(payload.Name)
	if payload.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	created, err := h.repo.Create(r.Context(), payload.Name)
	if err != nil {
		http.Error(w, "failed to create account", http.StatusInternalServerError)
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

	idStr := strings.TrimPrefix(r.URL.Path, "/accounts/")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid account id", http.StatusBadRequest)
		return
	}

	var payload struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	payload.Name = strings.TrimSpace(payload.Name)
	if payload.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	updated, err := h.repo.Update(r.Context(), id, payload.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "account not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to update account", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(updated)
}

// accountIDFromOpeningBalancePath extrait l'id depuis /accounts/{id}/opening-balance.
func accountIDFromOpeningBalancePath(path string) (uint64, error) {
	trimmed := strings.TrimPrefix(path, "/accounts/")
	trimmed = strings.TrimSuffix(trimmed, "/opening-balance")
	trimmed = strings.Trim(trimmed, "/")

	if trimmed == "" || strings.Contains(trimmed, "/") {
		return 0, strconv.ErrSyntax
	}

	return strconv.ParseUint(trimmed, 10, 64)
}

func (h *Handler) UpdateOpeningBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id, err := accountIDFromOpeningBalancePath(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid account id", http.StatusBadRequest)
		return
	}

	var payload struct {
		Amount float64 `json:"amount"`
		Date   string  `json:"date"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if payload.Date == "" {
		http.Error(w, "date is required", http.StatusBadRequest)
		return
	}

	updated, err := h.repo.SetOpeningBalance(r.Context(), id, payload.Amount, payload.Date)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "account not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to update opening balance", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(updated)
}

func (h *Handler) ClearOpeningBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id, err := accountIDFromOpeningBalancePath(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid account id", http.StatusBadRequest)
		return
	}

	updated, err := h.repo.ClearOpeningBalance(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "account not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to clear opening balance", http.StatusInternalServerError)
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

	idStr := strings.TrimPrefix(r.URL.Path, "/accounts/")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid account id", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "account not found", http.StatusNotFound)
			return
		}

		if err == ErrAccountInUse {
			http.Error(w, "ce compte est encore utilisé (achats, revenus ou catégories) — suppression impossible", http.StatusConflict)
			return
		}

		http.Error(w, "failed to delete account", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
