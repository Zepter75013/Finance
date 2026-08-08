package dbsettings

import (
	"encoding/json"
	"net/http"
	"strings"

	"finance/backend/internal/config"
)

// Handler expose le choix de moteur de base de données (MySQL ou SQLite
// locale) depuis l'écran Préférences. Ce choix ne peut prendre effet qu'au
// prochain démarrage du backend (la connexion est ouverte une seule fois,
// au lancement) — GET renvoie donc à la fois le moteur réellement actif sur
// ce process et celui enregistré pour le prochain démarrage, pour que le
// frontend puisse afficher "redémarrage nécessaire" si les deux diffèrent.
type Handler struct {
	// activeDriver est figé au démarrage de ce process (celui avec lequel il
	// s'est effectivement connecté) — jamais modifié en cours de route.
	activeDriver string
}

func NewHandler(activeDriver string) *Handler {
	return &Handler{activeDriver: activeDriver}
}

type databaseSettingsResponse struct {
	ActiveDriver     string `json:"active_driver"`
	ConfiguredDriver string `json:"configured_driver"`
	SQLitePath       string `json:"sqlite_path"`
	RestartRequired  bool   `json:"restart_required"`
}

func (h *Handler) currentResponse() databaseSettingsResponse {
	configuredDriver := h.activeDriver
	sqlitePath := ""

	if persisted, ok := config.LoadPersistedDBSettings(); ok {
		configuredDriver = persisted.Driver
		sqlitePath = persisted.SQLitePath
	}

	if sqlitePath == "" {
		sqlitePath = "data/finance.db"
	}

	return databaseSettingsResponse{
		ActiveDriver:     h.activeDriver,
		ConfiguredDriver: configuredDriver,
		SQLitePath:       sqlitePath,
		RestartRequired:  configuredDriver != h.activeDriver,
	}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(h.currentResponse())
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Driver string `json:"driver"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	driver := strings.ToLower(strings.TrimSpace(payload.Driver))
	if driver != config.DriverMySQL && driver != config.DriverSQLite {
		http.Error(w, "driver must be \"mysql\" or \"sqlite\"", http.StatusBadRequest)
		return
	}

	// Le chemin du fichier SQLite n'est pas configurable depuis l'UI (évite
	// tout risque de chemin arbitraire) — on garde celui déjà enregistré s'il
	// y en a un, sinon la valeur par défaut de l'app.
	sqlitePath := "data/finance.db"
	if persisted, ok := config.LoadPersistedDBSettings(); ok && persisted.SQLitePath != "" {
		sqlitePath = persisted.SQLitePath
	}

	if err := config.SavePersistedDBSettings(config.PersistedDBSettings{
		Driver:     driver,
		SQLitePath: sqlitePath,
	}); err != nil {
		http.Error(w, "échec de l'enregistrement du réglage", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(h.currentResponse())
}
