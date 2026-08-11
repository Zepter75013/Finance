package http

import (
	"net/http"

	"finance/backend/internal/auth"
	"finance/backend/internal/authctx"
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

func requireAuth(sessions *auth.Repository, next http.HandlerFunc) http.HandlerFunc {
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

		next.ServeHTTP(w, r)
	}
}
