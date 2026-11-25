package main

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
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
	r := chi.NewRouter()
	with := func(mw []httputil.Middleware, h http.Handler) http.Handler { return httputil.With(h, mw...) }
	use := func(mw []httputil.Middleware, m ...httputil.Middleware) []httputil.Middleware {
		next := make([]httputil.Middleware, len(mw)+len(m))
		copy(next, mw)
		copy(next[len(mw):], m)
		return next
	}

	r.NotFound(httputil.Status(http.StatusBadRequest, "Invalid address"))

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

	r.Route("/~", func(r chi.Router) {
		login := auth.Login(log, cypher, production)
		r.Method("GET", "/auth/login", with(mw, login))
		mw := use(mw, middleware.Logger)

		staticfs := with(mw, store.Handler(log, content.Static(), "/~"))
		r.Route("/admin", func(r chi.Router) {
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

			r.Method("GET", "/link", with(mw, link.Get()))
			r.Method("POST", "/link", with(mw, link.Post(log, cypher)))

			// NOTE: Fallback to parent staticfs
			r.Method("GET", "/*", staticfs)
		})

		r.Method("GET", "/*", staticfs)
	})

	mw = use(mw, middleware.Logger)
	if !production {
		r.Route("/test", func(r chi.Router) {
			r.Method("GET", "/countdown", with(mw, countdown.TestHandler(log, content.Static())))
		})
	}

	r.Method("GET", "/favicon.ico", with(mw, with(mw, store.Handler(log, content.Static(), "/"))))
	mw = use(mw,
		auth.Middleware(log, cypher),
		countdown.Middleware(log, cypher, content),
	)
	r.Method("GET", "/*", with(mw, store.Handler(log, content.Data(), "/")))
	return r
}
