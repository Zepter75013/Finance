package backup

import (
	"encoding/json"
	"os"
)

// settingsFile stores the user-chosen backup directory (and the automatic
// backup settings below) so they survive server restarts, independently of
// where the backups themselves live.
const settingsFile = "backup-settings.json"

const defaultRetentionDays = 30

type persistedSettings struct {
	Directory         string `json:"directory"`
	AutoBackupEnabled bool   `json:"auto_backup_enabled"`
	RetentionDays     int    `json:"retention_days"`
	// LastAutoBackupAt est en secondes Unix ; 0 signifie "jamais exécutée".
	LastAutoBackupAt int64 `json:"last_auto_backup_at"`
}

// loadPersistedSettings lit le fichier de réglages. Un fichier créé avant
// l'ajout de la sauvegarde automatique n'a pas les clés correspondantes : on
// applique alors les valeurs par défaut une seule fois et on réécrit le
// fichier, pour que tous les chargements suivants soient sans ambiguïté.
func loadPersistedSettings() (persistedSettings, bool) {
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		return persistedSettings{}, false
	}

	var s persistedSettings
	if err := json.Unmarshal(data, &s); err != nil || s.Directory == "" {
		return persistedSettings{}, false
	}

	var raw map[string]json.RawMessage
	_ = json.Unmarshal(data, &raw)
	if _, hasAutoKey := raw["auto_backup_enabled"]; !hasAutoKey {
		s.AutoBackupEnabled = true
		s.RetentionDays = defaultRetentionDays
		_ = savePersistedSettings(s)
	}

	if s.RetentionDays <= 0 {
		s.RetentionDays = defaultRetentionDays
	}

	return s, true
}

func savePersistedSettings(s persistedSettings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsFile, data, 0o644)
}
