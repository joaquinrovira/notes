package auth

import (
	_ "embed"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/joaquinrovira/notes/internal/auth/token"
	"github.com/joaquinrovira/notes/internal/httputil"
	"github.com/joaquinrovira/notes/internal/middleware"
)

func Middleware(t *token.Cypher) middleware.Middleware {
	Unauthorized := httputil.Status(http.StatusUnauthorized, "")
	Forbidden := httputil.Status(http.StatusForbidden, "Access to this resource has been denied.")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := token.Extract(t, r)
			if !ok {
				log.Println("missing auth payload")
				Unauthorized.ServeHTTP(w, r)
				return
			}

			if !allowed(payload, r.URL.Path) {
				log.Println("not allowed:", r.URL.Path)
				switch payload := payload.(type) {
				case *token.TokenV1:
					if payload.Verified {
						http.Redirect(w, r, payload.Home, http.StatusSeeOther)
						return
					}
				}
				Forbidden.ServeHTTP(w, r)
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
	if !token.Verified {
		return false
	}
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
