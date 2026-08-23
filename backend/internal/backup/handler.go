package backup

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxUploadBytes = 200 << 20 // 200 MB

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	files, err := h.service.List()
	if err != nil {
		http.Error(w, "échec de la lecture des sauvegardes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(files)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	file, err := h.service.Create(r.Context())
	if err != nil {
		http.Error(w, "échec de la création de la sauvegarde", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(file)
}

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	name, err := backupNameFromPath(r.URL.Path, "/download")
	if err != nil {
		http.Error(w, "nom de sauvegarde invalide", http.StatusBadRequest)
		return
	}

	file, meta, err := h.service.OpenForDownload(name)
	if err != nil {
		if errors.Is(err, ErrInvalidName) {
			http.Error(w, "nom de sauvegarde invalide", http.StatusBadRequest)
			return
		}

		http.Error(w, "sauvegarde introuvable", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/sql")
	w.Header().Set("Content-Disposition", `attachment; filename="`+meta.Name+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, file)
}

func (h *Handler) Restore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	name, err := backupNameFromPath(r.URL.Path, "/restore")
	if err != nil {
		http.Error(w, "nom de sauvegarde invalide", http.StatusBadRequest)
		return
	}

	if err := h.service.RestoreExisting(r.Context(), name); err != nil {
		if errors.Is(err, ErrInvalidName) {
			http.Error(w, "nom de sauvegarde invalide", http.StatusBadRequest)
			return
		}

		http.Error(w, "échec de la restauration : "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RestoreUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "fichier trop volumineux ou requête invalide", http.StatusBadRequest)
		return
	}

	uploaded, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "fichier manquant", http.StatusBadRequest)
		return
	}
	defer uploaded.Close()

	file, err := h.service.RestoreUpload(r.Context(), uploaded)
	if err != nil {
		http.Error(w, "échec de la restauration : "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(file)
}

type settingsPayload struct {
	Directory         string `json:"directory"`
	AutoBackupEnabled bool   `json:"auto_backup_enabled"`
	RetentionDays     int    `json:"retention_days"`
}

func (h *Handler) currentSettings() settingsPayload {
	return settingsPayload{
		Directory:         h.service.Directory(),
		AutoBackupEnabled: h.service.AutoBackupEnabled(),
		RetentionDays:     h.service.RetentionDays(),
	}
}

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(h.currentSettings())
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload settingsPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "requête invalide", http.StatusBadRequest)
		return
	}

	if err := h.service.SetDirectory(payload.Directory); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.SetAutoBackupSettings(payload.AutoBackupEnabled, payload.RetentionDays); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(h.currentSettings())
}

// PickDirectory opens a native macOS folder picker on the server's own
// desktop (this app runs locally, so that's the user's own screen) and
// returns the chosen path without saving it — the frontend still confirms
// via UpdateSettings.
func (h *Handler) PickDirectory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	dir, err := h.service.PickDirectoryDialog(ctx)
	if err != nil {
		if errors.Is(err, ErrDialogCancelled) {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(settingsPayload{Directory: dir})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	name, err := backupNameFromPath(r.URL.Path, "")
	if err != nil {
		http.Error(w, "nom de sauvegarde invalide", http.StatusBadRequest)
		return
	}

	if err := h.service.Delete(name); err != nil {
		if errors.Is(err, ErrInvalidName) {
			http.Error(w, "nom de sauvegarde invalide", http.StatusBadRequest)
			return
		}

		http.Error(w, "échec de la suppression", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// backupNameFromPath extracts the {name} segment from "/backups/{name}" or
// "/backups/{name}{suffix}" (e.g. suffix "/download" or "/restore").
func backupNameFromPath(path, suffix string) (string, error) {
	trimmed := strings.TrimPrefix(path, "/backups/")
	trimmed = strings.TrimSuffix(trimmed, suffix)
	trimmed = strings.Trim(trimmed, "/")

	if trimmed == "" || strings.Contains(trimmed, "/") {
		return "", ErrInvalidName
	}

	return trimmed, nil
}
