package store

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	httputil "github.com/joaquinrovira/notes/internal/util/http"
)

func Handler(log *slog.Logger, fs Filesystem, prefix string) http.Handler {
	NotFound := httputil.Status(http.StatusNotFound, "")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "" || path[0] != '/' {
			path = "/" + path
		}
		path = filepath.Clean(path)
		path = strings.TrimPrefix(path, prefix)

		if filepath.Base(path) == "index.html" {
			http.Redirect(w, r, filepath.Dir(path), http.StatusSeeOther)
			return
		}

		path = filepath.Join(".", path)

		// Check if adding ".html" to the file finds it
		func() {
			alt := path + ".html"
			_, err := fs.Stat(alt)
			if err == nil {
				path = alt
			}
		}()

		// Check if directory, if so, look for index.html
		info, err := fs.Stat(path)
		if err != nil {
			NotFound(w, r)
			return
		}
		if info.IsDir() {
			// Add trailing slashes to paths.
			if !strings.HasSuffix(r.URL.Path, "/") {
				http.Redirect(w, r, r.URL.Path+"/", http.StatusSeeOther)
				return
			}

			// Directories open index.html by default.
			path = filepath.Join(path, "index.html")
		}

		file, err := fs.OpenFile(path, os.O_RDONLY, os.ModePerm)
		if os.IsNotExist(err) {
			NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Error("open file", slog.String("path", path), slog.String("err", err.Error()))
			return
		}

		defer file.Close()
		http.ServeContent(w, r, filepath.Base(path), info.ModTime(), file)
	})
}
