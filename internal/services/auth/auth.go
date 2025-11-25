package auth

import (
	"net/http"

	"github.com/joaquinrovira/notes/internal/services/token"
)

const AuthCookie = "auth#data"

func Extract(Token *token.Service, r *http.Request) (token.Token, bool) {
	cookie, err := r.Cookie(AuthCookie)
	if err != nil {
		return nil, false
	}

	payload, err := Token.Decrypt(cookie.Value)
	if err != nil {
		return nil, false
	}

	return payload, true
}
