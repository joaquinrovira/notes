package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/joaquinrovira/notes/internal/auth/token"
	"github.com/joaquinrovira/notes/internal/config"
	"github.com/joaquinrovira/notes/internal/content"
)

func main() {
	ctx := context.Background()
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt, os.Kill)

	logger := func() *slog.Logger {
		switch strings.ToLower(config.OrDefault("LOG_FORMAT", "text")) {
		default:
			fallthrough
		case "text":
			return slog.New(slog.NewTextHandler(os.Stderr, nil))
		case "json":
			return slog.New(slog.NewJSONHandler(os.Stderr, nil))
		}
	}()

	err := run(ctx, logger)
	if err != nil {
		logger.Error(err.Error())
	}
}

func run(ctx context.Context, log *slog.Logger) error {
	production := strings.ToLower(os.Getenv("ENV")) == "production"
	if !production {
		log.Warn("Running in development mode.")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	username := os.Getenv("USERNAME")
	if username == "" {
		log.Warn("Using default admin username.")
		username = "admin"
	}

	password := os.Getenv("PASSWORD")
	if password == "" {
		log.Warn("Using default admin password.")
		password = "admin"
	}

	key, err := config.Require("ENCRYPTION_KEY")
	if err != nil {
		log.Error(err.Error())
		log.Warn("Using default dev key. Do not use in production.")
		key = "12345678901234567890123456789012"
	}

	t, err := token.New(key)
	if err != nil {
		return fmt.Errorf("token service: %v", err)
	}

	content, err := func() (content.Content, error) {
		switch strings.ToLower(os.Getenv("STORE")) {
		default:
			fallthrough
		case "git", "":
			return content.FromGit(ctx, log)
		case "fs":
			return content.FromFS()
		}
	}()
	if err != nil {
		return fmt.Errorf("content provider: %w", err)
	}

	mux := http.NewServeMux()
	Routes(
		ctx,
		log,
		mux,
		content,
		t,
		chimiddleware.BasicAuth("local", map[string]string{username: password}),
		production,
	)

	socket := "0.0.0.0:" + port
	log.Info("Starting", slog.String("Socket", socket))
	log.Info("Admin interface available", slog.String("URL", path.Join("localhost:"+port, "/auth/token")))
	return http.ListenAndServe(socket, mux)
}
