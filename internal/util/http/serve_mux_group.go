package httputil

import (
	"net/http"
	"path"
	"strings"
)

type ServeMuxRouter struct {
	pattern string
	mux     *http.ServeMux
}

func NewServeMuxRouter(mux *http.ServeMux) ServeMuxRouter {
	return ServeMuxRouter{mux: mux}
}

func (m ServeMuxRouter) Route(pattern string) ServeMuxRouter {
	m.pattern = joinPatternPath(m.Prefix(), pattern)
	return m
}

func (m ServeMuxRouter) Handle(pattern string, handler http.Handler) {
	rawPath := pattern
	prefix := ""
	if i := strings.LastIndex(pattern, " "); i != -1 {
		prefix = pattern[:i+1] // e.g. "GET "
		rawPath = pattern[i+1:]
	}

	fullPath := joinPatternPath(m.Prefix(), rawPath)
	m.mux.Handle(prefix+fullPath, handler)
}
func (m ServeMuxRouter) Prefix() string {
	return joinPatternPath("/", m.pattern)
}

func joinPatternPath(prefix, p string) string {
	hasTrailingSlash := strings.HasSuffix(p, "/")
	joined := path.Join("/", prefix, p)
	if hasTrailingSlash && !strings.HasSuffix(joined, "/") {
		joined += "/"
	}
	return joined
}
