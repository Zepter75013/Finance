package transfer

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

	accountID, err := strconv.ParseUint(r.URL.Query().Get("account_id"), 10, 64)
	if err != nil || accountID == 0 {
		http.Error(w, "account_id is required", http.StatusBadRequest)
		return
	}

	if !h.checkAccountAccess(w, r, accountID) {
		return
	}

	transfers, err := h.repo.ListByAccount(r.Context(), accountID)
	if err != nil {
		http.Error(w, "failed to load transfers", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(transfers)
}

func decodeAndValidate(w http.ResponseWriter, r *http.Request) (TransferInput, bool) {
	var input TransferInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return TransferInput{}, false
	}

	input.TransferDate = strings.TrimSpace(input.TransferDate)
	input.Note = strings.TrimSpace(input.Note)
	input.FromStatementReference = strings.TrimSpace(input.FromStatementReference)
	input.ToStatementReference = strings.TrimSpace(input.ToStatementReference)

	if input.FromAccountID == 0 || input.ToAccountID == 0 {
		http.Error(w, "from_account_id and to_account_id are required", http.StatusBadRequest)
		return TransferInput{}, false
	}

	if input.FromAccountID == input.ToAccountID {
		http.Error(w, "le compte source et le compte destination doivent être différents", http.StatusBadRequest)
		return TransferInput{}, false
	}

	if input.Amount <= 0 {
		http.Error(w, "amount must be greater than 0", http.StatusBadRequest)
		return TransferInput{}, false
	}

	if input.TransferDate == "" {
		http.Error(w, "transfer_date is required", http.StatusBadRequest)
		return TransferInput{}, false
	}

	return input, true
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	input, ok := decodeAndValidate(w, r)
	if !ok {
		return
	}

	if !h.checkAccountAccess(w, r, input.FromAccountID) {
		return
	}

	if !h.checkAccountAccess(w, r, input.ToAccountID) {
		return
	}

	created, err := h.repo.Create(r.Context(), input)
	if err != nil {
		http.Error(w, "failed to create transfer", http.StatusInternalServerError)
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

	id, err := transferIDFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid transfer id", http.StatusBadRequest)
		return
	}

	input, ok := decodeAndValidate(w, r)
	if !ok {
		return
	}

	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "transfer not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to update transfer", http.StatusInternalServerError)
		return
	}

	// Vérifie l'accès aux comptes actuels ET, si l'utilisateur déplace le
	// virement vers d'autres comptes, l'accès à ces nouveaux comptes aussi.
	if !h.checkAccountAccess(w, r, existing.FromAccountID) {
		return
	}
	if !h.checkAccountAccess(w, r, existing.ToAccountID) {
		return
	}
	if input.FromAccountID != existing.FromAccountID && !h.checkAccountAccess(w, r, input.FromAccountID) {
		return
	}
	if input.ToAccountID != existing.ToAccountID && !h.checkAccountAccess(w, r, input.ToAccountID) {
		return
	}

	// Changer le compte d'un côté déjà pointé attribuerait silencieusement ce
	// pointage à la mauvaise séquence de relevés — refusé plutôt que corrompu.
	if input.FromAccountID != existing.FromAccountID && existing.FromIsReconciled {
		http.Error(w, "impossible de changer le compte source d'un virement déjà pointé", http.StatusConflict)
		return
	}
	if input.ToAccountID != existing.ToAccountID && existing.ToIsReconciled {
		http.Error(w, "impossible de changer le compte destination d'un virement déjà pointé", http.StatusConflict)
		return
	}

	updated, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "transfer not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to update transfer", http.StatusInternalServerError)
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

	id, err := transferIDFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid transfer id", http.StatusBadRequest)
		return
	}

	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "transfer not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to delete transfer", http.StatusInternalServerError)
		return
	}

	if !h.checkAccountAccess(w, r, existing.FromAccountID) {
		return
	}
	if !h.checkAccountAccess(w, r, existing.ToAccountID) {
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "transfer not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to delete transfer", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func transferIDFromPath(path string) (uint64, error) {
	trimmed := strings.TrimPrefix(path, "/transfers/")
	trimmed = strings.Trim(trimmed, "/")

	if trimmed == "" || strings.Contains(trimmed, "/") {
		return 0, strconv.ErrSyntax
	}

	return strconv.ParseUint(trimmed, 10, 64)
}
