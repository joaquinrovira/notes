package countdown

import (
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"text/template"
	"time"

	"github.com/joaquinrovira/notes/internal/auth/token"
	"github.com/joaquinrovira/notes/internal/content"
	"github.com/joaquinrovira/notes/internal/middleware"
)

func New(t *token.Cypher, fs content.Filesystem) middleware.Middleware {
	countdown := countdown(t, fs)
	return func(next http.Handler) http.Handler {
		return token.WithToken(t, func(w http.ResponseWriter, r *http.Request, t token.Token) {
			switch token := t.(type) {
			case *token.TokenV1:
				if token.NotBefore != nil && time.Now().Before(*token.NotBefore) {
					countdown(w, r)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

//go:embed countdown.html.tmpl
var CountdownPage string

func countdown(t *token.Cypher, fs content.Filesystem) http.HandlerFunc {
	tmpl, err := template.New("countdown").Parse(CountdownPage)
	if err != nil {
		panic(err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		payload, ok := token.Extract(t, r)
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch token := payload.(type) {
		case *token.TokenV1:
			h, err := customCountdown(token, fs)
			if err != nil {
				log.Print(err)
				countdownToken(*token, tmpl, w)
				return
			}
			h.ServeHTTP(w, r)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func countdownToken(token token.TokenV1, tmpl *template.Template, w http.ResponseWriter) {
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

func customCountdown(token *token.TokenV1, fs content.Filesystem) (h http.Handler, err error) {
	location := path.Join(".", token.Home)
	if location == "" {
		return nil, fmt.Errorf("undefined home parameter")
	}

	stat, err := fs.Stat(location)
	if err != nil {
		return nil, fmt.Errorf("could not stat '%s': %w", location, err)
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("not a directory '%s'", location)
	}

	location = path.Join(location, "countdown.html")
	stat, err = fs.Stat(location)
	if err != nil {
		return nil, fmt.Errorf("could not stat '%s': %w", location, err)
	}

	file, err := fs.OpenFile(location, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("opening '%s': %w", location, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading '%s': %w", location, err)
	}

	tmpl, err := template.New("countdown-" + location).Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("reading '%s': invalid template: %w", location, err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}), nil
}
