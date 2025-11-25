package handlers

import (
	"log"
	"net/http"
	"path/filepath"

	"github.com/joaquinrovira/notes/internal/services/auth"
	"github.com/joaquinrovira/notes/internal/services/token"
)

type VerifyHandler struct {
	TokenService *token.Service
}

func (h *VerifyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract Token
	raw := r.URL.Query().Get("token")
	if raw == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}

	// Decrypt Payload
	payload, err := h.TokenService.Decrypt(raw)
	if err != nil {
		log.Printf("Verification Failed: %v", err)
		http.Error(w, "Invalid token", http.StatusBadRequest)
		return
	}

	switch token := payload.(type) {
	case *token.TokenV1:
		h.serveHTTPTokenV1(w, r, *token, raw)
		return
	}
}

func (h *VerifyHandler) serveHTTPTokenV1(w http.ResponseWriter, r *http.Request, token token.TokenV1, raw string) {
	// Set Cookie
	cookie := &http.Cookie{
		Name:     auth.AuthCookie,
		Value:    raw,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // TODO: true in production
		SameSite: http.SameSiteStrictMode,
	}

	// Align cookie expiry with token expiry if present
	if token.Expiration != nil {
		cookie.Expires = *token.Expiration
	}

	http.SetCookie(w, cookie)

	// Final Redirect
	location := filepath.Join("/", token.Index)
	http.Redirect(w, r, location, http.StatusSeeOther)
}
