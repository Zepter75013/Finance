package budget

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"finance/backend/internal/account"
	"finance/backend/internal/authctx"
	"finance/backend/internal/category"
)

type Handler struct {
	repo         *Repository
	categoryRepo *category.Repository
	accountRepo  *account.Repository
}

func NewHandler(repo *Repository, categoryRepo *category.Repository, accountRepo *account.Repository) *Handler {
	return &Handler{repo: repo, categoryRepo: categoryRepo, accountRepo: accountRepo}
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

	accountID, err := strconv.ParseUint(r.URL.Query().Get("account_id"), 10, 64)
	if err != nil || accountID == 0 {
		http.Error(w, "account_id is required", http.StatusBadRequest)
		return
	}

	monthKey := r.URL.Query().Get("month")
	if monthKey == "" {
		http.Error(w, "month is required", http.StatusBadRequest)
		return
	}

	if !h.checkAccountAccess(w, r, accountID) {
		return
	}

	budgets, err := h.repo.GetForAccount(r.Context(), accountID, monthKey)
	if err != nil {
		http.Error(w, "failed to load budgets", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(budgets)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		CategoryID uint64  `json:"category_id"`
		Month      string  `json:"month"`
		Amount     float64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if payload.CategoryID == 0 || payload.Month == "" {
		http.Error(w, "category_id and month are required", http.StatusBadRequest)
		return
	}

	if payload.Amount < 0 {
		http.Error(w, "le budget doit être un montant positif ou nul", http.StatusBadRequest)
		return
	}

	accountID, err := h.categoryRepo.AccountIDOf(r.Context(), payload.CategoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to update budget", http.StatusInternalServerError)
		return
	}

	if !h.checkAccountAccess(w, r, accountID) {
		return
	}

	budget, err := h.repo.Upsert(r.Context(), payload.CategoryID, payload.Month, payload.Amount)
	if err != nil {
		http.Error(w, "failed to update budget", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(budget)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	categoryID, err := strconv.ParseUint(r.URL.Query().Get("category_id"), 10, 64)
	if err != nil || categoryID == 0 {
		http.Error(w, "category_id is required", http.StatusBadRequest)
		return
	}

	monthKey := r.URL.Query().Get("month")
	if monthKey == "" {
		http.Error(w, "month is required", http.StatusBadRequest)
		return
	}

	accountID, err := h.categoryRepo.AccountIDOf(r.Context(), categoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "category not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to delete budget", http.StatusInternalServerError)
		return
	}

	if !h.checkAccountAccess(w, r, accountID) {
		return
	}

	if err := h.repo.Delete(r.Context(), categoryID, monthKey); err != nil {
		http.Error(w, "failed to delete budget", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
