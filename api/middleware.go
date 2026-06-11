package api

import (
	"crypto/subtle"
	"net/http"

	"https://github.com/PhotonProjects/Photon-Panel"
)

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, `{"error": "missing authorization"}`, http.StatusUnauthorized)
			return
		}

		// Support "Bearer <token>" and raw token
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		expected := config.Get().Panel.AuthToken
		if subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
			http.Error(w, `{"error": "invalid authorization"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
