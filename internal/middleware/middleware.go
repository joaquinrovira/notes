package middleware

import (
	"net/http"
	"slices"
)

type Middleware = func(next http.Handler) http.Handler

func Apply(handler http.Handler, middlewares ...Middleware) http.Handler {
	slices.Reverse(middlewares)
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}
	return handler
}
