package config

import "os"

type Config struct {
	AppPort         string
	DBDriver        string
	DBHost          string
	DBPort          string
	DBName          string
	DBUser          string
	DBPassword      string
	SQLitePath      string
	FrontendURL     string
	AdminUsername   string
	AdminPassword   string
	SMTPHost        string
	SMTPPort        string
	SMTPUsername    string
	SMTPPassword    string
	SMTPFrom        string
	BackupDir       string
	StatementPdfDir string
	ReportDir       string
}

func Load() Config {
	// Le choix mysql/sqlite persisté depuis l'écran Préférences (fichier
	// db-settings.json) prend le pas sur les variables d'environnement — mais
	// seulement s'il existe déjà, pour ne rien changer au comportement actuel
	// tant que personne n'a fait ce choix dans l'app.
	dbDriver := getEnv("DB_DRIVER", DriverMySQL)
	sqlitePath := getEnv("SQLITE_PATH", "data/finance.db")

	if persisted, ok := LoadPersistedDBSettings(); ok {
		dbDriver = persisted.Driver
		if persisted.SQLitePath != "" {
			sqlitePath = persisted.SQLitePath
		}
	}

	return Config{
		AppPort:         getEnv("APP_PORT", "8080"),
		DBDriver:        dbDriver,
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "3306"),
		DBName:          getEnv("DB_NAME", "finance"),
		DBUser:          getEnv("DB_USER", "root"),
		DBPassword:      getEnv("DB_PASSWORD", ""),
		SQLitePath:      sqlitePath,
		FrontendURL:     getEnv("FRONTEND_URL", "http://localhost:5173"),
		AdminUsername:   getEnv("ADMIN_USERNAME", ""),
		AdminPassword:   getEnv("ADMIN_PASSWORD", ""),
		SMTPHost:        getEnv("SMTP_HOST", ""),
		SMTPPort:        getEnv("SMTP_PORT", "587"),
		SMTPUsername:    getEnv("SMTP_USERNAME", ""),
		SMTPPassword:    getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:        getEnv("SMTP_FROM", ""),
		BackupDir:       getEnv("BACKUP_DIR", "backups"),
		StatementPdfDir: getEnv("STATEMENT_PDF_DIR", "statement_pdfs"),
		ReportDir:       getEnv("REPORT_DIR", "reports"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
