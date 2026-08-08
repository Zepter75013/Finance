package config

import "os"

type Config struct {
	AppPort         string
	DBHost          string
	DBPort          string
	DBName          string
	DBUser          string
	DBPassword      string
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
	return Config{
		AppPort:         getEnv("APP_PORT", "8080"),
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "3306"),
		DBName:          getEnv("DB_NAME", "finance"),
		DBUser:          getEnv("DB_USER", "root"),
		DBPassword:      getEnv("DB_PASSWORD", ""),
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
