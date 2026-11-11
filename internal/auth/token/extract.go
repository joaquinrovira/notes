package token

import (
	"encoding/json/v2"
	"log"
	"net/http"
)

const AuthCookie = "auth#data"

func Extract(t *Cypher, r *http.Request) (Token, bool) {
	cookie, err := r.Cookie(AuthCookie)
	if err != nil {
		return nil, false
	}

	payload, err := t.Decrypt(cookie.Value)
	if err != nil {
		return nil, false
	}

	return payload, true
}

func WithToken(t *Cypher, fn func(w http.ResponseWriter, r *http.Request, t Token)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, ok := Extract(t, r)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		p, _ := json.Marshal(payload)
		log.Println(string(p))
		fn(w, r, payload)
	})
}
