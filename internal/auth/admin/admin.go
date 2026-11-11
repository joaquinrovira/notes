package admin

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/joaquinrovira/notes/internal/auth/token"
)

//go:embed admin.get.html
var GetAdminPage string

//go:embed admin.post.html.tmpl
var PostAdminPage string

func Get() http.HandlerFunc {
	data := []byte(GetAdminPage)
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(data)
	}
}

func Post(t *token.Cypher) http.HandlerFunc {
	tmpl, err := template.New("token-gen").Parse(PostAdminPage)
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
				log.Println(err)
				return
			}
			nbf = &t
		}

		var naf *time.Time
		nafRaw := r.FormValue("naf")
		if nafRaw != "" {
			t, err := time.Parse("2006-01-02T15:04", nafRaw)
			if err != nil {
				log.Println(err)
				return
			}
			naf = &t
		}

		var exp *time.Time
		expRaw := r.FormValue("exp")
		if expRaw != "" {
			t, err := time.Parse("2006-01-02T15:04", expRaw)
			if err != nil {
				log.Println(err)
				return
			}
			exp = &t
		}

		indexRaw := r.FormValue("index")
		indexRaw = strings.TrimSpace(indexRaw)
		index := filepath.Clean(indexRaw)
		if index == "" {
			http.Error(w, "Index path is required", http.StatusBadRequest)
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
			http.Error(w, "Encryption failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		fullLink := fmt.Sprintf("/auth/login?token=%s", tokenStr)

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		type PostAdminData struct {
			Link string
		}
		err = tmpl.Execute(w, PostAdminData{Link: fullLink})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
