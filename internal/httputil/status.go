package httputil

import (
	_ "embed"
	"html/template"
	"net/http"
)

//go:embed status.html.tmpl
var status string

var statustpl *template.Template

func init() {
	tmpl, err := template.New("status").Parse(status)
	if err != nil {
		panic(err)
	}
	statustpl = tmpl
}

func Status(code int, message string) http.Handler {
	status := http.StatusText(code)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(code)
		type data struct {
			Status     int
			StatusText string
			Message    string
		}
		err := statustpl.Execute(w, data{code, status, message})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
}
