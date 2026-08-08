package incomes

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

	accountID, err := strconv.ParseUint(r.URL.Query().Get("account_id"), 10, 64)
	if err != nil || accountID == 0 {
		http.Error(w, "account_id is required", http.StatusBadRequest)
		return
	}

	incomes, err := h.repo.List(r.Context(), accountID)
	if err != nil {
		http.Error(w, "failed to load incomes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(incomes)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var input CreateIncomeInput

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	input.Source = strings.TrimSpace(input.Source)
	input.IncomeDate = strings.TrimSpace(input.IncomeDate)
	input.Note = strings.TrimSpace(input.Note)
	input.Reference = strings.TrimSpace(input.Reference)
	input.OperationLabel = strings.TrimSpace(input.OperationLabel)
	input.AdditionalInfo = strings.TrimSpace(input.AdditionalInfo)
	input.OperationType = strings.TrimSpace(input.OperationType)
	input.Category = strings.TrimSpace(input.Category)
	input.SubCategory = strings.TrimSpace(input.SubCategory)
	input.OperationDate = strings.TrimSpace(input.OperationDate)
	input.ValueDate = strings.TrimSpace(input.ValueDate)
	input.StatementReference = strings.TrimSpace(input.StatementReference)

	if input.Source == "" {
		http.Error(w, "source is required", http.StatusBadRequest)
		return
	}

	if input.AccountID == 0 {
		http.Error(w, "account_id is required", http.StatusBadRequest)
		return
	}

	if input.Amount <= 0 {
		http.Error(w, "amount must be greater than 0", http.StatusBadRequest)
		return
	}

	if input.IncomeDate == "" {
		http.Error(w, "income_date is required", http.StatusBadRequest)
		return
	}

	createdIncome, err := h.repo.Create(r.Context(), input)
	if err != nil {
		http.Error(w, "failed to create income", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(createdIncome)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id, err := incomeIDFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid income id", http.StatusBadRequest)
		return
	}

	var input CreateIncomeInput

	err = json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	input.Source = strings.TrimSpace(input.Source)
	input.IncomeDate = strings.TrimSpace(input.IncomeDate)
	input.Note = strings.TrimSpace(input.Note)
	input.Reference = strings.TrimSpace(input.Reference)
	input.OperationLabel = strings.TrimSpace(input.OperationLabel)
	input.AdditionalInfo = strings.TrimSpace(input.AdditionalInfo)
	input.OperationType = strings.TrimSpace(input.OperationType)
	input.Category = strings.TrimSpace(input.Category)
	input.SubCategory = strings.TrimSpace(input.SubCategory)
	input.OperationDate = strings.TrimSpace(input.OperationDate)
	input.ValueDate = strings.TrimSpace(input.ValueDate)
	input.StatementReference = strings.TrimSpace(input.StatementReference)

	if input.Source == "" {
		http.Error(w, "source is required", http.StatusBadRequest)
		return
	}

	if input.AccountID == 0 {
		http.Error(w, "account_id is required", http.StatusBadRequest)
		return
	}

	if input.Amount <= 0 {
		http.Error(w, "amount must be greater than 0", http.StatusBadRequest)
		return
	}

	if input.IncomeDate == "" {
		http.Error(w, "income_date is required", http.StatusBadRequest)
		return
	}

	updatedIncome, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "income not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to update income", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(updatedIncome)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id, err := incomeIDFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid income id", http.StatusBadRequest)
		return
	}

	err = h.repo.Delete(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "income not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to delete income", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func incomeIDFromPath(path string) (uint64, error) {
	trimmedPath := strings.TrimPrefix(path, "/incomes/")
	trimmedPath = strings.Trim(trimmedPath, "/")
	trimmedPath = strings.TrimSpace(trimmedPath)

	if trimmedPath == "" || strings.Contains(trimmedPath, "/") {
		return 0, strconv.ErrSyntax
	}

	return strconv.ParseUint(trimmedPath, 10, 64)
}
