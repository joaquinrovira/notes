package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/joaquinrovira/notes/internal/cache"
	"github.com/joaquinrovira/notes/internal/handlers"
	mdw "github.com/joaquinrovira/notes/internal/middleware"
	"github.com/joaquinrovira/notes/internal/services/token"
)

func main() {
	// 1. Config
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	username := os.Getenv("USERNAME")
	if username == "" {
		log.Println("WARNING: Using default admin username. Do not use in production.")
		username = "admin"
	}

	password := os.Getenv("PASSWORD")
	if password == "" {
		log.Println("WARNING: Using default admin password. Do not use in production.")
		password = "admin"
	}

	TokenService, err := token.NewServiceFromEnv()
	if err != nil {
		log.Fatalf("Failed to init token service: %v", err)
	}

	root, err := os.OpenRoot("./routes")
	if err != nil {
		panic(err)
	}

	static, err := os.OpenRoot("./static")
	if err != nil {
		panic(err)
	}

	middlewares := []mdw.Middleware{
		middleware.Recoverer,
		middleware.Throttle(10),
	}

	// 3. Routing
	mux := http.NewServeMux()

	// Auth Handlers
	verifyHandler := &handlers.VerifyHandler{TokenService: TokenService}
	mux.Handle("/auth/login", mdw.Apply(verifyHandler, middlewares...))

	middlewares = append(
		middlewares,
		middleware.Logger,
	)

	staticfs := mdw.Apply(cache.NewCachedFileServer(static), middlewares...)
	mux.Handle("/~/", http.StripPrefix("/~/", staticfs))
	mux.Handle("/favicon.ico", staticfs)

	adminMdw := append(
		middlewares,
		middleware.BasicAuth("local", map[string]string{username: password}),
	)
	mux.Handle("GET /auth/token", mdw.Apply(handlers.GetAdmin(), adminMdw...))
	mux.Handle("POST /auth/token", mdw.Apply(handlers.PostAdmin(TokenService), adminMdw...))

	mux.Handle("GET /countdown", mdw.Apply(handlers.Countdown(TokenService), middlewares...))

	middlewares = append(
		middlewares,
		mdw.Countdown(TokenService),
		mdw.Auth(TokenService),
	)
	mux.Handle("/", mdw.Apply(cache.NewCachedFileServer(root), middlewares...))

	// 4. Start Server
	log.Printf("Starting SecureFileServe on :%s", port)
	log.Printf("Admin Interface: http://localhost:%s/auth/token", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
