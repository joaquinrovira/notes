package admin

import (
	_ "embed"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/joaquinrovira/notes/internal/auth/token"
	"github.com/joaquinrovira/notes/internal/httputil"
)

//go:embed admin.get.html
var GetTokenGen string

//go:embed admin.post.html.tmpl
var PostTokenGen string

func Get() http.HandlerFunc {
	data := []byte(GetTokenGen)
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(data)
	}
}

func Post(log *slog.Logger, t *token.Cypher) http.HandlerFunc {
	log = log.With(slog.String("endpoint", "admin-post"))

	logFormErr := func(field string, err error) http.Handler {
		return httputil.Status(
			http.StatusBadRequest,
			fmt.Sprintf("form error: field %s: %s", field, err.Error()),
		)
	}

	tmpl, err := template.New("token-gen").Parse(PostTokenGen)
	if err != nil {
		panic(err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		var paths []string
		pathsRaw := r.FormValue("paths")
		pathsRaw = strings.TrimSpace(pathsRaw)
		if pathsRaw != "" {
			paths = strings.Split(pathsRaw, ",")
			for i, path := range paths {
				paths[i] = strings.TrimSpace(path)
			}
		}

		var nbf *time.Time
		nbfRaw := r.FormValue("nbf")
		if nbfRaw != "" {
			t, err := time.Parse("2006-01-02T15:04", nbfRaw)
			if err != nil {
				logFormErr("nbf", err).ServeHTTP(w, r)
				return
			}
			nbf = &t
		}

		var naf *time.Time
		nafRaw := r.FormValue("naf")
		if nafRaw != "" {
			t, err := time.Parse("2006-01-02T15:04", nafRaw)
			if err != nil {
				logFormErr("naf", err).ServeHTTP(w, r)
				return
			}
			naf = &t
		}

		var exp *time.Time
		expRaw := r.FormValue("exp")
		if expRaw != "" {
			t, err := time.Parse("2006-01-02T15:04", expRaw)
			if err != nil {
				logFormErr("exp", err).ServeHTTP(w, r)
				return
			}
			exp = &t
		}

		indexRaw := r.FormValue("index")
		indexRaw = strings.TrimSpace(indexRaw)
		index := filepath.Clean(indexRaw)
		if index == "" {
			logFormErr("index", fmt.Errorf("Index path is required")).ServeHTTP(w, r)
			return
		}

		payload := token.NewTokenV1()
		payload.NotBefore = nbf
		payload.NotAfter = naf
		payload.Expiration = exp
		payload.Paths = paths
		payload.Home = index

		tokenStr, err := t.Encrypt(payload)
		if err != nil {
			panic(fmt.Errorf("encryption failed: %w", err))
		}

		fullLink := fmt.Sprintf("/auth/login?token=%s", tokenStr)
		type PostAdminData struct{ Link string }
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		err = tmpl.Execute(w, PostAdminData{Link: fullLink})
		if err != nil {
			panic(err)
		}
	}
}
