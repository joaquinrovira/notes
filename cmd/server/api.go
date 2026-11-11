package main

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/joaquinrovira/notes/internal/auth"
	"github.com/joaquinrovira/notes/internal/auth/admin"
	"github.com/joaquinrovira/notes/internal/auth/token"
	"github.com/joaquinrovira/notes/internal/config"
	"github.com/joaquinrovira/notes/internal/countdown"
	"github.com/joaquinrovira/notes/internal/httputil"
	"github.com/joaquinrovira/notes/internal/store"
)

func routes(
	log *slog.Logger,
	cfg config.Config,
	mux *http.ServeMux,
	content store.Content,
	cypher *token.Cypher,
) {
	middlewares := []httputil.Middleware{
		middleware.Recoverer,
		middleware.ThrottleWithOpts(middleware.ThrottleOpts{
			Limit:          10,
			RetryAfterFn:   func(done bool) time.Duration { return 5 * time.Second },
			BacklogLimit:   100,
			BacklogTimeout: 5 * time.Second,
		}),
	}

	production := strings.ToLower(config.OrZero(cfg, "ENV")) == "production"
	if !production {
		log.Warn("running in development mode")
	}
	login := auth.Login(log, cypher, production)
	mux.Handle("/auth/login", httputil.With(login, middlewares...))

	if !production {
		mux.Handle("/test/countdown", httputil.With((countdown.TestHandler(log, content.Static()))))
	}

	middlewares = append(
		middlewares,
		middleware.Logger,
	)

	staticfs := httputil.With(store.Handler(log, content.Static()), middlewares...)
	mux.Handle("/~/", http.StripPrefix("/~/", staticfs))
	mux.Handle("/favicon.ico", staticfs)

	// Admin pages with basic auth
	func() {
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

		middlewares := append(
			middlewares,
			middleware.Throttle(4), // Stricter throttling for admin API
			middleware.BasicAuth(
				"Administration",
				map[string]string{username: password},
			),
		)
		mux.Handle("GET /auth/token", httputil.With(admin.Get(), middlewares...))
		mux.Handle("POST /auth/token", httputil.With(admin.Post(log, cypher), middlewares...))
	}()

	// Content pages with token auth
	func() {
		middlewares := append(
			middlewares,
			auth.Middleware(log, cypher),
			countdown.New(log, cypher, content.Data()),
		)
		mux.Handle("/", httputil.With(store.Handler(log, content.Data()), middlewares...))
	}()
}
