package category

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func normalizeType(raw string) (Type, bool) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))

	if trimmed == "" {
		return TypeAchat, true
	}

	switch Type(trimmed) {
	case TypeAchat, TypeRevenu:
		return Type(trimmed), true
	default:
		return "", false
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	accountID, err := strconv.ParseUint(r.URL.Query().Get("account_id"), 10, 64)
	if err != nil || accountID == 0 {
		http.Error(w, "account_id is required", http.StatusBadRequest)
		return
	}

	categories, err := h.repo.List(r.Context(), accountID)
	if err != nil {
		http.Error(w, "failed to load categories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(categories)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Name      string `json:"name"`
		Type      string `json:"type"`
		AccountID uint64 `json:"account_id"`
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

	if payload.AccountID == 0 {
		http.Error(w, "account_id is required", http.StatusBadRequest)
		return
	}

	categoryType, ok := normalizeType(payload.Type)
	if !ok {
		http.Error(w, "invalid category type", http.StatusBadRequest)
		return
	}

	category, err := h.repo.Create(r.Context(), payload.AccountID, payload.Name, categoryType)
	if err != nil {
		http.Error(w, "failed to create category", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(category)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/categories/")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid category id", http.StatusBadRequest)
		return
	}

	var payload struct {
		Name string `json:"name"`
		Type string `json:"type"`
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

	categoryType, ok := normalizeType(payload.Type)
	if !ok {
		http.Error(w, "invalid category type", http.StatusBadRequest)
		return
	}

	category, err := h.repo.Update(r.Context(), id, payload.Name, categoryType)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to update category", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(category)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/categories/")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid category id", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to delete category", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
