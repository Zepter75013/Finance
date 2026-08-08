package config

import (
	"encoding/json"
	"os"
)

// Le choix de base de données (MySQL ou SQLite locale) doit être connu
// AVANT toute connexion — il ne peut donc pas être stocké dans l'une des
// deux bases elles-mêmes (problème de l'œuf et la poule). Il vit dans un
// petit fichier JSON à côté du binaire, à la manière de backup-settings.json
// pour le dossier de sauvegarde.
const dbSettingsFile = "db-settings.json"

const (
	DriverMySQL  = "mysql"
	DriverSQLite = "sqlite"
)

type PersistedDBSettings struct {
	Driver     string `json:"driver"`
	SQLitePath string `json:"sqlite_path"`
}

func LoadPersistedDBSettings() (PersistedDBSettings, bool) {
	data, err := os.ReadFile(dbSettingsFile)
	if err != nil {
		return PersistedDBSettings{}, false
	}

	var settings PersistedDBSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return PersistedDBSettings{}, false
	}

	if settings.Driver != DriverMySQL && settings.Driver != DriverSQLite {
		return PersistedDBSettings{}, false
	}

	return settings, true
}

func SavePersistedDBSettings(settings PersistedDBSettings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(dbSettingsFile, data, 0o644)
}
