package countdown

import (
	_ "embed"
	"log/slog"
	"net/http"
	"text/template"
	"time"

	"github.com/joaquinrovira/notes/internal/auth/token"
	"github.com/joaquinrovira/notes/internal/store"
	httputil "github.com/joaquinrovira/notes/internal/util/http"
)

func Middleware(log *slog.Logger, t *token.Cypher, content store.Content) httputil.Middleware {
	countdown := Handler(log, t, content)
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

func RenderCountdown(token token.TokenV1, tmpl *template.Template, w http.ResponseWriter) {
	var countdown int64
	if token.NotBefore != nil {
		countdown = token.NotBefore.Unix()
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	type Data struct {
		Location    string
		UnixSeconds int64
	}
	err := tmpl.Execute(w, Data{Location: token.Home, UnixSeconds: countdown})
	if err != nil {
		panic(err)
	}
}

//go:embed countdown.html.tmpl
var CountdownPage string

func Default(fs store.Filesystem) (t *template.Template) {
	t, err := CountdownInDir(fs, ".")
	if err == nil {
		return t
	}

	t, err = template.New("countdown").Parse(CountdownPage)
	if err != nil {
		panic(err)
	}

	return t
}
