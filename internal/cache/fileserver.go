package cache

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// cacheEntry holds the content and modification time
type cacheEntry struct {
	content []byte
	modTime time.Time
}

// CachedFileServer serves files from disk but caches them in memory
type CachedFileServer struct {
	os *os.Root
	// cache maps file paths to cacheEntry
	cache sync.Map
	// ttl defines how long an item stays valid (simplified for this spec)
	ttl time.Duration
}

func NewCachedFileServer(root *os.Root) *CachedFileServer {
	return &CachedFileServer{
		os:  root,
		ttl: 5 * time.Minute,
	}
}

func (fs *CachedFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := "/" + filepath.Join(".", filepath.Clean(r.URL.Path))

	if filepath.Base(path) == "index.html" {
		http.Redirect(w, r, filepath.Dir(path)+"/", http.StatusSeeOther)
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
		if strings.HasSuffix(r.URL.Path, "/") {
			http.Redirect(w, r, strings.TrimSuffix(r.URL.Path, "/"), http.StatusSeeOther)
			return
		}
		path = filepath.Join(path, "index.html")
	}

	// 1. Check Cache
	if item, ok := fs.cache.Load(path); ok {
		entry := item.(cacheEntry)
		// Serve from memory using ServeContent (handles Range requests, ETag, etc.)
		http.ServeContent(w, r, filepath.Base(path), entry.modTime, bytes.NewReader(entry.content))
		return
	}

	// 2. Cache Miss: Read from Disk
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

	// Get fresh stat for ModTime
	info, err = fs.os.Stat(path)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	// 3. Store in Cache
	// Note: In a production app, we would check file size before caching to prevent OOM
	fs.cache.Store(path, cacheEntry{
		content: content,
		modTime: info.ModTime(),
	})

	log.Printf("Cache Miss: Loaded %s into memory", path)

	// 4. Serve
	http.ServeContent(w, r, filepath.Base(path), info.ModTime(), bytes.NewReader(content))
}
