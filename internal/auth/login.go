package auth

import (
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"github.com/joaquinrovira/notes/internal/auth/token"
	"github.com/joaquinrovira/notes/internal/httputil"
)

func Login(log *slog.Logger, t *token.Cypher, secure bool) http.Handler {
	Missing := httputil.Status(http.StatusBadRequest, "Missing authorization request token.")
	Invalid := httputil.Status(http.StatusBadRequest, "The received authorization token is not valid.")
	Expired := httputil.Status(http.StatusGone, "Oops! This link has expired.")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract Token
		raw := r.URL.Query().Get("token")
		if raw == "" {
			Missing.ServeHTTP(w, r)
			return
		}

		// Decrypt Payload
		payload, err := t.Decrypt(raw)
		if err != nil {
			log.Debug("token verification failed", slog.String("Err", err.Error()))
			Invalid.ServeHTTP(w, r)
			return
		}

		// Handle login (token version dependent)
		switch token := payload.(type) {
		case *token.TokenV1:
			serveHTTPTokenV1(log, t, token, secure, Expired).ServeHTTP(w, r)
			return
		}
	})
}

func serveHTTPTokenV1(
	log *slog.Logger,
	t *token.Cypher,
	v *token.TokenV1,
	secure bool,
	Expired http.Handler,
) http.Handler {
	if v.NotAfter != nil {
		if time.Now().After(*v.NotAfter) {
			return Expired
		}
	}

	next := *v
	next.Verified = true
	next.NotAfter = nil

	payload, err := t.Encrypt(&next)
	if err != nil {
		log.Error("failed to encrypt token", slog.String("Err", err.Error()))
		return httputil.Status(http.StatusInternalServerError, "")
	}

	cookie := &http.Cookie{
		Name:     token.AuthCookie,
		Value:    payload,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}

	if v.Expiration != nil {
		cookie.Expires = *v.Expiration
	}

	// Set cookie and redirect to home URL
	location := filepath.Join("/", v.Home)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, cookie)
		http.Redirect(w, r, location, http.StatusSeeOther)
	})
}
