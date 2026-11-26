package main

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/joaquinrovira/notes/internal/auth"
	"github.com/joaquinrovira/notes/internal/auth/admin/link"
	"github.com/joaquinrovira/notes/internal/auth/token"
	"github.com/joaquinrovira/notes/internal/config"
	"github.com/joaquinrovira/notes/internal/countdown"
	"github.com/joaquinrovira/notes/internal/store"
	httputil "github.com/joaquinrovira/notes/internal/util/http"
)

func routes(
	log *slog.Logger,
	cfg config.Config,
	content store.Content,
	cypher *token.Cypher,
) http.Handler {
	var mw []httputil.Middleware
	mux := http.NewServeMux()
	r := httputil.NewServeMuxRouter(mux)
	with := func(mw []httputil.Middleware, h http.Handler) http.Handler { return httputil.With(h, mw...) }
	use := func(mw []httputil.Middleware, m ...httputil.Middleware) []httputil.Middleware {
		next := make([]httputil.Middleware, len(mw)+len(m))
		copy(next, mw)
		copy(next[len(mw):], m)
		return next
	}

	mw = use(mw,
		middleware.RequestID,
		middleware.ClientIPFromHeader("CF-Connecting-IP"),
		middleware.Recoverer,
		middleware.ThrottleWithOpts(middleware.ThrottleOpts{
			Limit:          10,
			RetryAfterFn:   func(done bool) time.Duration { return 5 * time.Second },
			BacklogLimit:   100,
			BacklogTimeout: 5 * time.Second,
		}),
	)

	production := strings.ToLower(config.OrZero(cfg, "ENV")) == "production"
	if !production {
		log.Warn("running in development mode")
	}

	staticfs := store.Handler(log, content.Static(), "")

	func() {
		r := r.Route("/~/")

		login := auth.Login(log, cypher, production)
		r.Handle("GET /auth/login", with(mw, login))
		mw := use(mw, middleware.Logger)
		r.Handle("GET /", http.StripPrefix("/~/", staticfs))
		func() {
			r := r.Route("/admin")

			username := config.OrZero(cfg, "USERNAME")
			if username == "" {
				log.Warn("using default admin username")
				username = "admin"
			}
			password := config.OrZero(cfg, "PASSWORD")
			if password == "" {
				log.Warn("using default admin password")
				password = "admin"
			}

			mw := use(mw,
				middleware.Throttle(4), // Stricter throttling for admin
				middleware.BasicAuth(
					"Administration",
					map[string]string{username: password},
				),
			)

			r.Handle("GET /link", with(mw, link.Get()))
			r.Handle("POST /link", with(mw, link.Post(log, cypher)))
		}()
	}()

	mw = use(mw, middleware.Logger)
	func() {
		if production {
			return
		}
		r := r.Route("/test")

		r.Handle("GET /countdown", with(mw, countdown.TestHandler(log, content.Static())))
	}()

	r.Handle("GET /favicon.ico", with(mw, staticfs))
	mw = use(mw,
		auth.Middleware(log, cypher),
		countdown.Middleware(log, cypher, content),
	)
	r.Handle("GET /", with(mw, store.Handler(log, content.Data(), "/")))
	return mux
}
