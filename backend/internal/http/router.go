package http

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"finance/backend/internal/account"
	"finance/backend/internal/auth"
	"finance/backend/internal/backup"
	"finance/backend/internal/category"
	"finance/backend/internal/config"
	"finance/backend/internal/dbsettings"
	"finance/backend/internal/incomes"
	"finance/backend/internal/mailer"
	"finance/backend/internal/purchase"
	"finance/backend/internal/report"
	"finance/backend/internal/statement"
	"finance/backend/internal/subcategory"
	"finance/backend/internal/user"
)

func NewRouter(db *sql.DB, cfg config.Config) (http.Handler, error) {
	mux := http.NewServeMux()

	backupService, err := backup.NewService(cfg, cfg.BackupDir)
	if err != nil {
		return nil, err
	}
	backupHandler := backup.NewHandler(backupService)

	dbSettingsHandler := dbsettings.NewHandler(cfg.DBDriver)

	accountRepo := account.NewRepository(db)
	accountHandler := account.NewHandler(accountRepo)

	categoryRepo := category.NewRepository(db)
	categoryHandler := category.NewHandler(categoryRepo, accountRepo)

	subCategoryRepo := subcategory.NewRepository(db)
	subCategoryHandler := subcategory.NewHandler(subCategoryRepo, accountRepo)

	purchaseRepo := purchase.NewRepository(db)
	purchaseHandler := purchase.NewHandler(purchaseRepo, accountRepo)

	incomeRepo := incomes.NewRepository(db)
	incomeHandler := incomes.NewHandler(incomeRepo, accountRepo)

	statementRepo := statement.NewRepository(db)
	statementHandler := statement.NewHandler(statementRepo, cfg.StatementPdfDir, accountRepo)

	reportService, err := report.NewService(cfg.ReportDir)
	if err != nil {
		return nil, err
	}
	reportHandler := report.NewHandler(reportService)

	userRepo := user.NewRepository(db)
	userHandler := user.NewHandler(userRepo)
	sessionRepo := auth.NewRepository(db)
	mailerClient := mailer.New(cfg)
	authHandler := auth.NewHandler(sessionRepo, userRepo, mailerClient)

	mux.HandleFunc("/health", healthHandler(cfg.DBDriver))

	mux.HandleFunc("/auth/login", authHandler.Login)
	mux.HandleFunc("/auth/logout", authHandler.Logout)
	mux.HandleFunc("/auth/me", authHandler.Me)
	mux.HandleFunc("/auth/change-password", requireAuth(sessionRepo, authHandler.ChangePassword))
	mux.HandleFunc("/auth/request-reset-code", authHandler.RequestResetCode)
	mux.HandleFunc("/auth/reset-password-with-code", authHandler.ResetPasswordWithCode)

	mux.HandleFunc("/accounts", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			accountHandler.List(w, r)
		case http.MethodPost:
			accountHandler.Create(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/accounts/", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			accountHandler.Update(w, r)
		case http.MethodDelete:
			accountHandler.Delete(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/categories", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			categoryHandler.List(w, r)
		case http.MethodPost:
			categoryHandler.Create(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/categories/", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			categoryHandler.Update(w, r)
		case http.MethodDelete:
			categoryHandler.Delete(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/subcategories", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			subCategoryHandler.List(w, r)
		case http.MethodPost:
			subCategoryHandler.Create(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/subcategories/", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			subCategoryHandler.Update(w, r)
		case http.MethodDelete:
			subCategoryHandler.Delete(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/purchases", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			purchaseHandler.List(w, r)
		case http.MethodPost:
			purchaseHandler.Create(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/purchases/", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			purchaseHandler.Update(w, r)
		case http.MethodDelete:
			purchaseHandler.Delete(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/incomes", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			incomeHandler.List(w, r)
		case http.MethodPost:
			incomeHandler.Create(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/incomes/", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			incomeHandler.Update(w, r)
		case http.MethodDelete:
			incomeHandler.Delete(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/statements", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			statementHandler.List(w, r)
		case http.MethodPost:
			statementHandler.Upsert(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/statements/", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pdfs") {
			hasPdfID := strings.Contains(r.URL.Path, "/pdfs/")

			switch {
			case r.Method == http.MethodGet && !hasPdfID:
				statementHandler.ListPdfs(w, r)
			case r.Method == http.MethodPost && !hasPdfID:
				statementHandler.UploadPdf(w, r)
			case r.Method == http.MethodGet && hasPdfID:
				statementHandler.DownloadPdf(w, r)
			case r.Method == http.MethodDelete && hasPdfID:
				statementHandler.DeletePdf(w, r)
			case r.Method == http.MethodOptions:
				w.WriteHeader(http.StatusNoContent)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return
		}

		switch r.Method {
		case http.MethodPut:
			statementHandler.SetLocked(w, r)
		case http.MethodDelete:
			statementHandler.Delete(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/users", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			userHandler.List(w, r)
		case http.MethodPost:
			userHandler.Create(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/users/", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			userHandler.Update(w, r)
		case http.MethodDelete:
			userHandler.Delete(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/backups", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			backupHandler.List(w, r)
		case http.MethodPost:
			backupHandler.Create(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/backups/restore-upload", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			backupHandler.RestoreUpload(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/backups/settings", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			backupHandler.GetSettings(w, r)
		case http.MethodPut:
			backupHandler.UpdateSettings(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/backups/settings/pick-directory", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			backupHandler.PickDirectory(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/settings/database", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			dbSettingsHandler.Get(w, r)
		case http.MethodPut:
			dbSettingsHandler.Update(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/backups/", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			backupHandler.Download(w, r)
		case http.MethodPost:
			backupHandler.Restore(w, r)
		case http.MethodDelete:
			backupHandler.Delete(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/reports", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			reportHandler.Upload(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/reports/latest", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			reportHandler.GetLatestMetadata(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/reports/latest/pdf", requireAuth(sessionRepo, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			reportHandler.DownloadLatestPdf(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	return withCORS(cfg.FrontendURL, mux), nil
}

// healthHandler expose aussi le moteur de base de données actif (mysql ou
// sqlite) — une information publique, sans rapport avec les identifiants de
// connexion, utile pour l'afficher sur l'écran de connexion (avant
// authentification, donc inaccessible à /settings/database).
func healthHandler(dbDriver string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		response := map[string]string{
			"status":    "ok",
			"service":   "finance-backend",
			"timestamp": time.Now().Format(time.RFC3339),
			"db_driver": dbDriver,
		}

		_ = json.NewEncoder(w).Encode(response)
	}
}
