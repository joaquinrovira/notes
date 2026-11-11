package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/joaquinrovira/notes/internal/auth"
	"github.com/joaquinrovira/notes/internal/auth/admin"
	"github.com/joaquinrovira/notes/internal/auth/token"
	"github.com/joaquinrovira/notes/internal/content"
	"github.com/joaquinrovira/notes/internal/countdown"
	"github.com/joaquinrovira/notes/internal/middleware"
)

func Routes(
	ctx context.Context,
	log *slog.Logger,
	mux *http.ServeMux,
	c content.Content,
	t *token.Cypher,
	basicAuth middleware.Middleware,
	production bool,
) {
	middlewares := []middleware.Middleware{
		chimiddleware.Recoverer,
		chimiddleware.ThrottleWithOpts(chimiddleware.ThrottleOpts{
			Limit:          10,
			RetryAfterFn:   func(ctxDone bool) time.Duration { return 5 * time.Second },
			BacklogLimit:   100,
			BacklogTimeout: 5 * time.Second,
		}),
	}

	login := auth.Login(log, t, production)
	mux.Handle("/auth/login", middleware.Apply(login, middlewares...))

	middlewares = append(
		middlewares,
		chimiddleware.Logger,
	)

	staticfs := middleware.Apply(content.Handler(c.Static()), middlewares...)
	mux.Handle("/~/", http.StripPrefix("/~/", staticfs))
	mux.Handle("/favicon.ico", staticfs)

	// Admin pages with basic auth
	func() {
		middlewares := append(
			middlewares,
			chimiddleware.Throttle(4), // Stricter throttling for admin API
			basicAuth,
		)
		mux.Handle("GET /auth/token", middleware.Apply(admin.Get(), middlewares...))
		mux.Handle("POST /auth/token", middleware.Apply(admin.Post(t), middlewares...))
	}()

	// Content pages with token auth
	func() {
		middlewares := append(
			middlewares,
			auth.Middleware(t),
			countdown.New(t, c.Data()),
		)
		mux.Handle("/", middleware.Apply(content.Handler(c.Data()), middlewares...))
	}()
}
