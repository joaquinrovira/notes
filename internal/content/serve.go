package content

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func Handler(fs Filesystem) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "" || path[0] != '/' {
			path = "/" + path
		}
		path = filepath.Clean(path)

		if filepath.Base(path) == "index.html" {
			http.Redirect(w, r, filepath.Dir(path), http.StatusSeeOther)
			return
		}

		// Check if directory, if so, look for index.html
		path = filepath.Join(".", path)
		info, err := fs.Stat(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if info.IsDir() {
			// Remove trailing slashes from paths.
			if r.URL.Path != "/" && strings.HasSuffix(r.URL.Path, "/") {
				http.Redirect(w, r, strings.TrimSuffix(r.URL.Path, "/"), http.StatusSeeOther)
				return
			}
			// Directories open index.html by default.
			path = filepath.Join(path, "index.html")
		}

		file, err := fs.OpenFile(path, os.O_RDONLY, os.ModePerm)
		if err != nil {
			if os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		http.ServeContent(w, r, filepath.Base(path), info.ModTime(), file)
	})
}
