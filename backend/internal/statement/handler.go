package statement

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"finance/backend/internal/account"
	"finance/backend/internal/authctx"
)

const maxPdfUploadBytes = 30 << 20 // 30 MB

type Handler struct {
	repo        *Repository
	pdfDir      string
	accountRepo *account.Repository
}

func NewHandler(repo *Repository, pdfDir string, accountRepo *account.Repository) *Handler {
	return &Handler{repo: repo, pdfDir: pdfDir, accountRepo: accountRepo}
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

	statements, err := h.repo.List(r.Context(), accountID)
	if err != nil {
		http.Error(w, "failed to load statements", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(statements)
}

func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var input UpsertStatementInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input.Reference = strings.TrimSpace(input.Reference)
	input.StatementDate = strings.TrimSpace(input.StatementDate)
	input.PeriodStart = strings.TrimSpace(input.PeriodStart)
	input.PeriodEnd = strings.TrimSpace(input.PeriodEnd)

	if input.Reference == "" {
		http.Error(w, "le numéro de relevé est obligatoire", http.StatusBadRequest)
		return
	}

	if input.AccountID == 0 {
		http.Error(w, "le compte est obligatoire", http.StatusBadRequest)
		return
	}

	if !h.checkAccountAccess(w, r, input.AccountID) {
		return
	}

	saved, err := h.repo.Upsert(r.Context(), input)
	if err != nil {
		if err == ErrStatementLocked {
			http.Error(w, "ce relevé est verrouillé — déverrouille-le avant de le modifier", http.StatusConflict)
			return
		}

		http.Error(w, "échec de l'enregistrement du relevé", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(saved)
}

func (h *Handler) SetLocked(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id, err := statementIDFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid statement id", http.StatusBadRequest)
		return
	}

	var input struct {
		IsLocked bool `json:"is_locked"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "relevé introuvable", http.StatusNotFound)
			return
		}

		http.Error(w, "échec de la mise à jour du relevé", http.StatusInternalServerError)
		return
	}

	if !h.checkAccountAccess(w, r, existing.AccountID) {
		return
	}

	updated, err := h.repo.SetLocked(r.Context(), id, input.IsLocked)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "relevé introuvable", http.StatusNotFound)
			return
		}

		http.Error(w, "échec de la mise à jour du relevé", http.StatusInternalServerError)
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

	id, err := statementIDFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid statement id", http.StatusBadRequest)
		return
	}

	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "relevé introuvable", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to delete statement", http.StatusInternalServerError)
		return
	}

	if !h.checkAccountAccess(w, r, existing.AccountID) {
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "relevé introuvable", http.StatusNotFound)
			return
		}

		if err == ErrStatementLocked {
			http.Error(w, "ce relevé est verrouillé — déverrouille-le avant de le supprimer", http.StatusConflict)
			return
		}

		http.Error(w, "failed to delete statement", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func statementIDFromPath(path string) (uint64, error) {
	return statementIDFromPathWithSuffix(path, "")
}

func statementIDFromPathWithSuffix(path, suffix string) (uint64, error) {
	trimmedPath := strings.TrimPrefix(path, "/statements/")
	trimmedPath = strings.TrimSuffix(trimmedPath, suffix)
	trimmedPath = strings.Trim(trimmedPath, "/")
	trimmedPath = strings.TrimSpace(trimmedPath)

	if trimmedPath == "" || strings.Contains(trimmedPath, "/") {
		return 0, strconv.ErrSyntax
	}

	return strconv.ParseUint(trimmedPath, 10, 64)
}

func (h *Handler) pdfPath(diskFilename string) string {
	return filepath.Join(h.pdfDir, diskFilename)
}

func statementIDFromPdfsPath(path string) (uint64, error) {
	trimmed := strings.TrimPrefix(path, "/statements/")

	idx := strings.Index(trimmed, "/pdfs")
	if idx == -1 {
		return 0, strconv.ErrSyntax
	}

	return strconv.ParseUint(trimmed[:idx], 10, 64)
}

func pdfIDFromPdfsPath(path string) (uint64, error) {
	const marker = "/pdfs/"

	idx := strings.Index(path, marker)
	if idx == -1 {
		return 0, strconv.ErrSyntax
	}

	rest := strings.Trim(path[idx+len(marker):], "/")
	if rest == "" || strings.Contains(rest, "/") {
		return 0, strconv.ErrSyntax
	}

	return strconv.ParseUint(rest, 10, 64)
}

func (h *Handler) ListPdfs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	statementID, err := statementIDFromPdfsPath(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid statement id", http.StatusBadRequest)
		return
	}

	pdfs, err := h.repo.ListPdfs(r.Context(), statementID)
	if err != nil {
		http.Error(w, "échec du chargement des fichiers", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(pdfs)
}

// UploadPdf attaches one more scanned PDF to a statement — a relevé can have
// several (recto/verso, plusieurs pages scannées séparément, etc). It's
// allowed even on a locked statement — storing a source document isn't
// "editing" the reconciliation data itself.
func (h *Handler) UploadPdf(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	statementID, err := statementIDFromPdfsPath(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid statement id", http.StatusBadRequest)
		return
	}

	if _, err := h.repo.FindByID(r.Context(), statementID); err != nil {
		http.Error(w, "relevé introuvable", http.StatusNotFound)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPdfUploadBytes)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "fichier trop volumineux ou requête invalide", http.StatusBadRequest)
		return
	}

	uploaded, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "fichier manquant", http.StatusBadRequest)
		return
	}
	defer uploaded.Close()

	if !strings.EqualFold(filepath.Ext(header.Filename), ".pdf") {
		http.Error(w, "seuls les fichiers PDF sont acceptés", http.StatusBadRequest)
		return
	}

	if err := os.MkdirAll(h.pdfDir, 0o755); err != nil {
		http.Error(w, "échec du stockage du fichier", http.StatusInternalServerError)
		return
	}

	diskFilename := fmt.Sprintf("%d_%d.pdf", statementID, time.Now().UnixNano())
	destPath := h.pdfPath(diskFilename)

	destFile, err := os.Create(destPath)
	if err != nil {
		http.Error(w, "échec du stockage du fichier", http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(destFile, uploaded); err != nil {
		destFile.Close()
		os.Remove(destPath)
		http.Error(w, "échec du stockage du fichier", http.StatusInternalServerError)
		return
	}
	destFile.Close()

	originalFilename := header.Filename
	if originalFilename == "" {
		originalFilename = diskFilename
	}

	created, err := h.repo.AddPdf(r.Context(), statementID, diskFilename, originalFilename)
	if err != nil {
		os.Remove(destPath)
		http.Error(w, "échec de l'enregistrement du fichier", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(created)
}

// pdfForStatement loads a PDF row and checks it actually belongs to the
// statement named in the URL, so one relevé's request can't reach into
// another's files by guessing a pdf id.
func (h *Handler) pdfForStatement(ctx context.Context, path string) (StatementPdf, error) {
	statementID, err := statementIDFromPdfsPath(path)
	if err != nil {
		return StatementPdf{}, err
	}

	pdfID, err := pdfIDFromPdfsPath(path)
	if err != nil {
		return StatementPdf{}, err
	}

	pdf, err := h.repo.FindPdf(ctx, pdfID)
	if err != nil {
		return StatementPdf{}, err
	}

	if pdf.StatementID != statementID {
		return StatementPdf{}, sql.ErrNoRows
	}

	return pdf, nil
}

func (h *Handler) DownloadPdf(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	pdf, err := h.pdfForStatement(r.Context(), r.URL.Path)
	if err != nil {
		http.Error(w, "PDF introuvable", http.StatusNotFound)
		return
	}

	diskFilename, err := h.repo.PdfDiskFilename(r.Context(), pdf.ID)
	if err != nil {
		http.Error(w, "PDF introuvable", http.StatusNotFound)
		return
	}

	file, err := os.Open(h.pdfPath(diskFilename))
	if err != nil {
		http.Error(w, "PDF introuvable", http.StatusNotFound)
		return
	}
	defer file.Close()

	displayName := pdf.OriginalFilename
	if displayName == "" {
		displayName = diskFilename
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, displayName))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, file)
}

func (h *Handler) DeletePdf(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	pdf, err := h.pdfForStatement(r.Context(), r.URL.Path)
	if err != nil {
		http.Error(w, "PDF introuvable", http.StatusNotFound)
		return
	}

	diskFilename, err := h.repo.PdfDiskFilename(r.Context(), pdf.ID)
	if err == nil {
		if err := os.Remove(h.pdfPath(diskFilename)); err != nil && !errors.Is(err, os.ErrNotExist) {
			http.Error(w, "échec de la suppression du fichier", http.StatusInternalServerError)
			return
		}
	}

	if err := h.repo.DeletePdf(r.Context(), pdf.ID); err != nil {
		http.Error(w, "échec de la suppression du fichier", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
