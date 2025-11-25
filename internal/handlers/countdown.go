package handlers

import (
	_ "embed"
	"net/http"
	"text/template"

	"github.com/joaquinrovira/notes/internal/services/auth"
	"github.com/joaquinrovira/notes/internal/services/token"
)

//go:embed countdown.html.tmpl
var CountdownPage string

type CountdownData struct {
	Location    string
	UnixSeconds int64
}

func Countdown(TokenService *token.Service) http.HandlerFunc {
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
			countdownTokenV1(*token, tmpl, w, r)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func countdownTokenV1(token token.TokenV1, tmpl *template.Template, w http.ResponseWriter, r *http.Request) {
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
