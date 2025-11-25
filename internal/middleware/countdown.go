package middleware

import (
	"net/http"
	"time"

	"github.com/joaquinrovira/notes/internal/services/auth"
	"github.com/joaquinrovira/notes/internal/services/token"
)

func Countdown(TokenService *token.Service) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next := func() {
				next.ServeHTTP(w, r)
			}

			payload, ok := auth.Extract(TokenService, r)
			if !ok {
				next()
				return
			}

			switch token := payload.(type) {
			case *token.TokenV1:
				if token.NotBefore != nil && time.Now().Before(*token.NotBefore) {
					http.Redirect(w, r, "/countdown", http.StatusSeeOther)
					return
				}
			}

			next()
		})
	}
}
