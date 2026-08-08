package database

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func OpenSQLite(path string) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Appli mono-utilisateur, faible concurrence : une seule connexion
	// physique évite complètement les erreurs "database is locked" plutôt
	// que de les gérer via des retries — plus simple et suffisant ici.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// PRAGMA appliqué une seule fois puisqu'il n'y a qu'une connexion
	// physique (MaxOpenConns=1) — fiable, pas besoin de le refaire à chaque
	// requête ni de passer par un paramètre de connexion.
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		return nil, err
	}

	for _, statement := range sqliteSchemaStatements {
		if _, err := db.Exec(statement); err != nil {
			return nil, err
		}
	}

	return db, nil
}
