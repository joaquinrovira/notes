package handlers

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/joaquinrovira/notes/internal/services/token"
)

//go:embed admin.html
var GetAdminPage string

func GetAdmin() http.HandlerFunc {
	data := []byte(GetAdminPage)
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(data)
	}
}

func PostAdmin(TokenService *token.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
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

		payload := token.NewTokenV1()
		payload.NotBefore = nbf
		payload.Expiration = exp
		payload.Paths = paths
		payload.Index = "/test"

		tokenStr, err := TokenService.Encrypt(payload)
		if err != nil {
			http.Error(w, "Encryption failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		fullLink := fmt.Sprintf("/auth/verify?token=%s", tokenStr)

		html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<body>
	<h1>Link Generated</h1>
	<p>Share this link:</p>
	<textarea style="width:100%%; height: 100px;">%s</textarea>
	<br><br>
	<a href="/auth/generate">Generate Another</a>
</body>
</html>
	`, fullLink)

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}
}
