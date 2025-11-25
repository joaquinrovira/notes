package handlers

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/joaquinrovira/notes/internal/services/token"
)

//go:embed admin.get.html
var GetAdminPage string

//go:embed admin.post.html.tmpl
var PostAdminPage string

type PostAdminData struct {
	Link string
}

func GetAdmin() http.HandlerFunc {
	data := []byte(GetAdminPage)
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(data)
	}
}

func PostAdmin(TokenService *token.Service) http.HandlerFunc {
	tmpl, err := template.New("token-gen").Parse(PostAdminPage)
	if err != nil {
		panic(err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		indexRaw := r.FormValue("index")
		pathsRaw := r.FormValue("paths")
		nbfRaw := r.FormValue("nbf")
		expRaw := r.FormValue("exp")

		var paths []string
		pathsRaw = strings.TrimSpace(pathsRaw)
		if pathsRaw != "" {
			paths = strings.Split(pathsRaw, ",")
			for i, path := range paths {
				paths[i] = strings.TrimSpace(path)
			}
		}

		var nbf *time.Time
		if nbfRaw != "" {
			t, err := time.Parse("2006-01-02T15:04", nbfRaw)
			if err != nil {
				log.Println(err)
				return
			}
			nbf = &t
		}

		var exp *time.Time
		if expRaw != "" {
			t, err := time.Parse("2006-01-02T15:04", expRaw)
			if err != nil {
				log.Println(err)
				return
			}
			exp = &t
		}

		indexRaw = strings.TrimSpace(indexRaw)
		if indexRaw == "" {
			http.Error(w, "Index path is required", http.StatusBadRequest)
			return
		}

		payload := token.NewTokenV1()
		payload.NotBefore = nbf
		payload.Expiration = exp
		payload.Paths = paths
		payload.Index = filepath.Clean(indexRaw)

		tokenStr, err := TokenService.Encrypt(payload)
		if err != nil {
			http.Error(w, "Encryption failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		fullLink := fmt.Sprintf("/auth/login?token=%s", tokenStr)

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		err = tmpl.Execute(w, PostAdminData{Link: fullLink})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
