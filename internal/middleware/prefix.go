package middleware

import (
	"net/http"
	"net/url"
	"path"
)

func AddPrefix(prefix string) Middleware {
	if prefix == "" {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := path.Join(prefix, r.URL.Path)
			rp := path.Join(prefix, r.URL.RawPath)
			r2 := new(http.Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.Path = p
			r2.URL.RawPath = rp
			next.ServeHTTP(w, r2)
		})
	}
}
