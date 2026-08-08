package report

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const maxReportUploadBytes = 30 << 20 // 30 MB

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Upload receives the PDF generated client-side and stores it as the new
// "latest" report, replacing whatever was there before.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxReportUploadBytes)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "fichier trop volumineux ou requête invalide", http.StatusBadRequest)
		return
	}

	uploaded, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "fichier manquant", http.StatusBadRequest)
		return
	}
	defer uploaded.Close()

	criteria := r.FormValue("criteria")

	meta, err := h.service.SaveLatest(criteria, uploaded)
	if err != nil {
		http.Error(w, "échec de l'enregistrement du rapport", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(meta)
}

func (h *Handler) GetLatestMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	meta, err := h.service.GetLatestMetadata()
	if err != nil {
		if errors.Is(err, ErrNoReport) {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		http.Error(w, "échec de la lecture du rapport", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(meta)
}

func (h *Handler) DownloadLatestPdf(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	file, err := h.service.OpenLatestPdf()
	if err != nil {
		if errors.Is(err, ErrNoReport) {
			http.Error(w, "aucun rapport généré pour l'instant", http.StatusNotFound)
			return
		}

		http.Error(w, "échec de la lecture du rapport", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="rapport-finance-ui.pdf"`)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, file)
}
