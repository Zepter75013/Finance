package database

import (
	"database/sql"
	"fmt"

	"finance/backend/internal/config"
)

// Open ouvre la base configurée (MySQL par défaut, ou SQLite locale si
// choisi depuis l'écran Préférences) — point d'entrée unique utilisé par
// cmd/api, pour que le reste de l'app (repositories, handlers) reste
// entièrement agnostique du moteur réellement utilisé.
func Open(cfg config.Config) (*sql.DB, error) {
	switch cfg.DBDriver {
	case config.DriverSQLite:
		return OpenSQLite(cfg.SQLitePath)
	case config.DriverMySQL, "":
		return OpenMySQL(cfg)
	default:
		return nil, fmt.Errorf("unknown DB_DRIVER %q", cfg.DBDriver)
	}
}
