package recurring

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"finance/backend/internal/account"
	"finance/backend/internal/authctx"
)

type Handler struct {
	repo        *Repository
	accountRepo *account.Repository
	service     *Service
}

func NewHandler(repo *Repository, accountRepo *account.Repository, service *Service) *Handler {
	return &Handler{repo: repo, accountRepo: accountRepo, service: service}
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

func normalizeType(raw string) (string, bool) {
	switch raw {
	case TypeAchat, TypeRevenu:
		return raw, true
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

	if !h.checkAccountAccess(w, r, accountID) {
		return
	}

	items, err := h.repo.ListByAccount(r.Context(), accountID)
	if err != nil {
		http.Error(w, "failed to load recurring transactions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(items)
}

func decodeInput(r *http.Request) (CreateRecurringInput, error) {
	var input CreateRecurringInput
	err := json.NewDecoder(r.Body).Decode(&input)
	input.Merchant = strings.TrimSpace(input.Merchant)
	input.Source = strings.TrimSpace(input.Source)
	input.PaymentMethod = strings.TrimSpace(input.PaymentMethod)
	input.OperationType = strings.TrimSpace(input.OperationType)
	input.Category = strings.TrimSpace(input.Category)
	input.SubCategory = strings.TrimSpace(input.SubCategory)
	input.Note = strings.TrimSpace(input.Note)
	return input, err
}

func validateInput(input CreateRecurringInput) string {
	if input.AccountID == 0 {
		return "account_id is required"
	}

	if _, ok := normalizeType(input.Type); !ok {
		return "type must be 'achat' or 'revenu'"
	}

	if _, err := time.Parse("2006-01-02", input.FirstRunDate); err != nil {
		return "first_run_date must be a valid date (YYYY-MM-DD)"
	}

	if input.Amount <= 0 {
		return "amount must be greater than 0"
	}

	if input.Type == TypeAchat {
		if input.Merchant == "" {
			return "merchant is required for achat"
		}
		if input.PaymentMethod == "" {
			return "payment_method is required for achat"
		}
		if input.CategoryID == 0 {
			return "category_id is required for achat"
		}
	}

	if input.Type == TypeRevenu && input.Source == "" {
		return "source is required for revenu"
	}

	return ""
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	input, err := decodeInput(r)
	if err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if msg := validateInput(input); msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if !h.checkAccountAccess(w, r, input.AccountID) {
		return
	}

	// Une récurrence est active par défaut à la création — le champ n'existe
	// que pour permettre de la mettre en pause plus tard via Update.
	input.IsActive = true

	created, err := h.repo.Create(r.Context(), input)
	if err != nil {
		http.Error(w, "failed to create recurring transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func recurringIDFromPath(path string) (uint64, error) {
	trimmed := strings.TrimPrefix(path, "/recurring/")
	trimmed = strings.Trim(trimmed, "/")

	if trimmed == "" || strings.Contains(trimmed, "/") {
		return 0, strconv.ErrSyntax
	}

	return strconv.ParseUint(trimmed, 10, 64)
}

func recurringIDFromExecutePath(path string) (uint64, error) {
	trimmed := strings.TrimPrefix(path, "/recurring/")
	trimmed = strings.Trim(trimmed, "/")
	trimmed = strings.TrimSuffix(trimmed, "execute")
	trimmed = strings.Trim(trimmed, "/")

	if trimmed == "" || strings.Contains(trimmed, "/") {
		return 0, strconv.ErrSyntax
	}

	return strconv.ParseUint(trimmed, 10, 64)
}

// Execute réalise manuellement une occurrence en retard (échéance déjà
// passée) — utilisé quand l'utilisateur veut rattraper mois par mois sans
// attendre le prochain passage du planificateur de fond.
func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id, err := recurringIDFromExecutePath(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid recurring transaction id", http.StatusBadRequest)
		return
	}

	existing, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "recurring transaction not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to execute recurring transaction", http.StatusInternalServerError)
		return
	}

	if !h.checkAccountAccess(w, r, existing.AccountID) {
		return
	}

	updated, err := h.service.ExecuteNow(r.Context(), id)
	if err != nil {
		if err == ErrNotDue {
			http.Error(w, "cette récurrence n'est pas encore due", http.StatusConflict)
			return
		}

		http.Error(w, "failed to execute recurring transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(updated)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id, err := recurringIDFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid recurring transaction id", http.StatusBadRequest)
		return
	}

	var payload struct {
		CreateRecurringInput
		IsActive *bool `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	input := payload.CreateRecurringInput
	input.Merchant = strings.TrimSpace(input.Merchant)
	input.Source = strings.TrimSpace(input.Source)
	input.PaymentMethod = strings.TrimSpace(input.PaymentMethod)
	input.OperationType = strings.TrimSpace(input.OperationType)
	input.Category = strings.TrimSpace(input.Category)
	input.SubCategory = strings.TrimSpace(input.SubCategory)
	input.Note = strings.TrimSpace(input.Note)

	if msg := validateInput(input); msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	existing, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "recurring transaction not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to update recurring transaction", http.StatusInternalServerError)
		return
	}

	if !h.checkAccountAccess(w, r, existing.AccountID) {
		return
	}

	if input.AccountID != existing.AccountID && !h.checkAccountAccess(w, r, input.AccountID) {
		return
	}

	if payload.IsActive != nil {
		input.IsActive = *payload.IsActive
	} else {
		input.IsActive = existing.IsActive
	}

	updated, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		http.Error(w, "failed to update recurring transaction", http.StatusInternalServerError)
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

	id, err := recurringIDFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid recurring transaction id", http.StatusBadRequest)
		return
	}

	existing, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "recurring transaction not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to delete recurring transaction", http.StatusInternalServerError)
		return
	}

	if !h.checkAccountAccess(w, r, existing.AccountID) {
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "recurring transaction not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to delete recurring transaction", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
