package middleware

import (
	_ "embed"
	"encoding/json/v2"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/joaquinrovira/notes/internal/auth/token"
)

func Countdown(TokenService *token.Cypher) Middleware {
	countdown := countdown(TokenService)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next := func() { next.ServeHTTP(w, r) }

			payload, ok := token.Extract(TokenService, r)
			if !ok {
				next()
				return
			}

			p, _ := json.Marshal(payload)
			log.Println(string(p))

			switch token := payload.(type) {
			case *token.TokenV1:
				if token.NotBefore != nil && time.Now().Before(*token.NotBefore) {
					countdown(w, r)
					return
				}
			}

			next()
		})
	}
}

//go:embed countdown.html.tmpl
var CountdownPage string

func countdown(TokenService *token.Cypher) http.HandlerFunc {
	tmpl, err := template.New("countdown").Parse(CountdownPage)
	if err != nil {
		panic(err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		payload, ok := token.Extract(TokenService, r)
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch token := payload.(type) {
		case *token.TokenV1:
			countdownTokenV1(*token, tmpl, w)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func countdownTokenV1(token token.TokenV1, tmpl *template.Template, w http.ResponseWriter) {
	var countdown int64
	if token.NotBefore != nil {
		countdown = token.NotBefore.Unix()
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	type CountdownData struct {
		Location    string
		UnixSeconds int64
	}
	err := tmpl.Execute(w, CountdownData{Location: token.Home, UnixSeconds: countdown})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
