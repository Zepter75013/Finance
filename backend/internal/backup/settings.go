package backup

import (
	"encoding/json"
	"os"
)

// settingsFile stores the user-chosen backup directory so it survives
// server restarts, independently of where the backups themselves live.
const settingsFile = "backup-settings.json"

type persistedSettings struct {
	Directory string `json:"directory"`
}

func loadPersistedDirectory() (string, bool) {
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		return "", false
	}

	var s persistedSettings
	if err := json.Unmarshal(data, &s); err != nil || s.Directory == "" {
		return "", false
	}

	return s.Directory, true
}

func savePersistedDirectory(dir string) error {
	data, err := json.MarshalIndent(persistedSettings{Directory: dir}, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsFile, data, 0o644)
}
