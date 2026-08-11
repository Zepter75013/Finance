package subcategory

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"finance/backend/internal/account"
	"finance/backend/internal/authctx"
)

type Handler struct {
	repo        *Repository
	accountRepo *account.Repository
}

func NewHandler(repo *Repository, accountRepo *account.Repository) *Handler {
	return &Handler{repo: repo, accountRepo: accountRepo}
}

// checkAccountAccess renvoie false (et écrit déjà la réponse 403) si
// l'utilisateur courant n'a pas le droit d'agir sur accountID.
func (h *Handler) checkAccountAccess(w http.ResponseWriter, r *http.Request, accountID uint64) bool {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}

	allowed, err := h.accountRepo.UserCanAccess(r.Context(), userID, accountID)
	if err != nil {
		http.Error(w, "échec de la vérification des droits", http.StatusInternalServerError)
		return false
	}

	if !allowed {
		http.Error(w, "vous n'avez pas accès à ce compte", http.StatusForbidden)
		return false
	}

	return true
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

	categoryAccountID, err := h.repo.AccountIDOfCategory(r.Context(), payload.CategoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to create sub-category", http.StatusInternalServerError)
		return
	}

	if !h.checkAccountAccess(w, r, categoryAccountID) {
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

	existingAccountID, err := h.repo.AccountIDOf(r.Context(), id)
	if err != nil {
		http.Error(w, "failed to update sub-category", http.StatusInternalServerError)
		return
	}

	if !h.checkAccountAccess(w, r, existingAccountID) {
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

	existingAccountID, err := h.repo.AccountIDOf(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "sub-category not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to delete sub-category", http.StatusInternalServerError)
		return
	}

	if !h.checkAccountAccess(w, r, existingAccountID) {
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
