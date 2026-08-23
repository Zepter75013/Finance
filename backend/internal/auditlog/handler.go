package auditlog

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

const listLimit = 200

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

	entries, err := h.repo.ListRecent(r.Context(), listLimit)
	if err != nil {
		http.Error(w, "failed to load audit log", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(entries)
}

// entityLabels traduit le premier segment d'un chemin d'API en libellé
// français lisible dans le journal — à défaut d'une correspondance connue,
// le segment brut est utilisé tel quel plutôt que de masquer l'information.
var entityLabels = map[string]string{
	"purchases":     "achat",
	"incomes":       "revenu",
	"categories":    "catégorie",
	"subcategories": "sous-catégorie",
	"accounts":      "compte",
	"statements":    "relevé",
	"recurring":     "récurrence",
	"users":         "utilisateur",
	"backups":       "sauvegarde",
	"settings":      "paramètre",
}

// DeriveEntity extrait un (type, id) lisible depuis le chemin d'une requête
// — ex: "/purchases/123" -> ("achat", "123"), "/recurring/123/execute" ->
// ("récurrence", "123"), "/purchases" -> ("achat", "") pour une création où
// l'id n'est connu qu'après coup.
func DeriveEntity(path string) (entityType string, entityID string) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) == 0 || segments[0] == "" {
		return "", ""
	}

	entityType = segments[0]
	if label, ok := entityLabels[entityType]; ok {
		entityType = label
	}

	if len(segments) > 1 {
		if _, err := strconv.ParseUint(segments[1], 10, 64); err == nil {
			entityID = segments[1]
		}
	}

	return entityType, entityID
}
