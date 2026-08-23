package http

import (
	"context"
	"log"
	"net/http"

	"finance/backend/internal/auditlog"
	"finance/backend/internal/auth"
	"finance/backend/internal/authctx"
	"finance/backend/internal/user"
)

func withCORS(frontendURL string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", frontendURL)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// statusRecorder mémorise le code de statut écrit par le handler, pour que
// le journal d'audit sache si l'action a réellement réussi (le
// http.ResponseWriter standard ne permet pas de le relire après coup).
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(status int) {
	rec.status = status
	rec.ResponseWriter.WriteHeader(status)
}

func requireAuth(sessions *auth.Repository, userRepo *user.Repository, auditRepo *auditlog.Repository, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(auth.CookieName)
		if err != nil {
			http.Error(w, "non authentifié", http.StatusUnauthorized)
			return
		}

		session, err := sessions.FindValidSession(r.Context(), cookie.Value)
		if err != nil {
			http.Error(w, "non authentifié", http.StatusUnauthorized)
			return
		}

		r = r.WithContext(authctx.WithUserID(r.Context(), session.UserID))

		// Seules les mutations sont journalisées — jamais les simples
		// consultations (GET), qui n'ont pas leur place dans un journal
		// d'audit et en noieraient rapidement le contenu utile.
		if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodDelete {
			next.ServeHTTP(w, r)
			return
		}

		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)

		method := r.Method
		path := r.URL.Path
		userID := session.UserID
		statusCode := recorder.status

		go func() {
			username := "?"
			if account, err := userRepo.FindByID(context.Background(), userID); err == nil {
				username = account.Username
			}

			entityType, entityID := auditlog.DeriveEntity(path)

			entry := auditlog.Entry{
				UserID:     &userID,
				Username:   username,
				Method:     method,
				Path:       path,
				EntityType: entityType,
				EntityID:   entityID,
				StatusCode: statusCode,
			}

			if err := auditRepo.Create(context.Background(), entry); err != nil {
				log.Printf("audit: échec de l'enregistrement (%s %s): %v", method, path, err)
			}
		}()
	}
}
