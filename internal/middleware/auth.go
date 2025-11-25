package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/joaquinrovira/notes/internal/services/auth"
	"github.com/joaquinrovira/notes/internal/services/token"
)

func Auth(TokenService *token.Service) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := auth.Extract(TokenService, r)
			if !ok {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			if !allowed(payload, r.URL.Path) {
				w.WriteHeader(http.StatusLocked)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func allowed(playload token.Token, location string) bool {
	switch t := playload.(type) {
	case *token.TokenV1:
		return allowTokenV1(*t, location)
	}
	return false
}

func allowTokenV1(token token.TokenV1, location string) bool {
	now := time.Now()

	if token.NotBefore != nil {
		if now.Before(*token.NotBefore) {
			return false
		}
	}

	if token.Expiration != nil {
		if now.After(*token.Expiration) {
			return false
		}
	}

	for _, pattern := range token.Paths {
		if strings.HasPrefix(location, pattern) {
			return true
		}
	}
	return false
}
