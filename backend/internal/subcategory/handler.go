package subcategory

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

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	subCategories, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, "failed to load sub-categories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(subCategories)
}

// Create is idempotent with respect to (category_id, name): importing the
// same pair twice returns the existing row instead of erroring, since the
// primary use case is bulk CSV import where duplicates must be skipped.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		CategoryID uint64 `json:"category_id"`
		Name       string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	payload.Name = strings.TrimSpace(payload.Name)

	if payload.CategoryID == 0 {
		http.Error(w, "category_id is required", http.StatusBadRequest)
		return
	}

	if payload.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	existing, err := h.repo.FindByCategoryAndName(r.Context(), payload.CategoryID, payload.Name)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(existing)
		return
	} else if err != sql.ErrNoRows {
		http.Error(w, "failed to create sub-category", http.StatusInternalServerError)
		return
	}

	created, err := h.repo.Create(r.Context(), payload.CategoryID, payload.Name)
	if err != nil {
		http.Error(w, "failed to create sub-category", http.StatusInternalServerError)
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

	idStr := strings.TrimPrefix(r.URL.Path, "/subcategories/")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid sub-category id", http.StatusBadRequest)
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

	current, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "sub-category not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to update sub-category", http.StatusInternalServerError)
		return
	}

	existing, err := h.repo.FindByCategoryAndName(r.Context(), current.CategoryID, payload.Name)
	if err == nil && existing.ID != id {
		http.Error(w, "une sous-catégorie avec ce nom existe déjà", http.StatusConflict)
		return
	} else if err != nil && err != sql.ErrNoRows {
		http.Error(w, "failed to update sub-category", http.StatusInternalServerError)
		return
	}

	updated, err := h.repo.Update(r.Context(), id, payload.Name)
	if err != nil {
		http.Error(w, "failed to update sub-category", http.StatusInternalServerError)
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

	idStr := strings.TrimPrefix(r.URL.Path, "/subcategories/")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid sub-category id", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "sub-category not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to delete sub-category", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
