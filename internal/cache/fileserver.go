package cache

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type FileServer struct {
	os *os.Root
}

func NewCachedFileServer(root *os.Root) *FileServer {
	return &FileServer{os: root}
}

func (fs *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	info, err := fs.os.Stat(filepath.Join(".", path))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if info.IsDir() {
		if r.URL.Path != "/" && strings.HasSuffix(r.URL.Path, "/") {
			http.Redirect(w, r, strings.TrimSuffix(r.URL.Path, "/"), http.StatusSeeOther)
			return
		}
		path = filepath.Join(path, "index.html")
	}

	file, err := fs.os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	http.ServeContent(w, r, filepath.Base(path), info.ModTime(), bytes.NewReader(content))
}
