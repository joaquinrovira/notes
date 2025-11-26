package middleware

import (
	_ "embed"
	"encoding/json"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/joaquinrovira/notes/internal/services/auth"
	"github.com/joaquinrovira/notes/internal/services/token"
)

func Countdown(TokenService *token.Service) Middleware {
	countdown := countdown(TokenService)
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

type CountdownData struct {
	Location    string
	UnixSeconds int64
}

func countdown(TokenService *token.Service) http.HandlerFunc {
	tmpl, err := template.New("countdown").Parse(CountdownPage)
	if err != nil {
		panic(err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		payload, ok := auth.Extract(TokenService, r)
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
	err := tmpl.Execute(w, CountdownData{Location: token.Index, UnixSeconds: countdown})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
