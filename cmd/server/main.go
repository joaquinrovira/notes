package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/joaquinrovira/notes/internal/auth/token"
	"github.com/joaquinrovira/notes/internal/config"
	"github.com/joaquinrovira/notes/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
}
func run() error {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt, os.Kill)

	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	log := logger(cfg)

	cypher, err := cypher(log, cfg)
	if err != nil {
		return fmt.Errorf("token service: %v", err)
	}

	content, err := content(ctx, log, cfg, wg)
	if err != nil {
		return fmt.Errorf("content provider: %w", err)
	}

	listen := api(ctx, log, cfg, cypher, content)
	return listen()
}

func logger(cfg config.Config) *slog.Logger {
	raw := config.OrDefault(cfg, "LOG_LEVEL", slog.LevelDebug.String())
	leveler := new(slog.Level)
	err := leveler.UnmarshalText([]byte(raw))
	if err != nil {
		leveler = new(slog.LevelDebug)
	}
	opts := &slog.HandlerOptions{Level: leveler}

	switch strings.ToLower(config.OrDefault(cfg, "LOG_FORMAT", "text")) {
	default:
		fallthrough
	case "text":
		return slog.New(slog.NewTextHandler(os.Stderr, opts))
	case "json":
		return slog.New(slog.NewJSONHandler(os.Stderr, opts))
	}
}

func cypher(
	log *slog.Logger,
	cfg config.Config,
) (*token.Cypher, error) {
	key, err := config.Require(cfg, "ENCRYPTION_KEY")
	if err != nil {
		log.Error(err.Error())
		log.Warn("using default dev key - do not use in production")
		key = "12345678901234567890123456789012"
	}
	return token.NewCypher(key)
}

func content(
	ctx context.Context,
	log *slog.Logger,
	cfg config.Config,
	wg *sync.WaitGroup,
) (store.Content, error) {
	log = log.With("service", "storage")
	switch strings.ToLower(os.Getenv("STORE")) {
	default:
		fallthrough
	case "git", "":
		log = log.With("backend", "git")
		log.Info("sourcing content from git store")
		return store.FromGit(ctx, log, cfg, wg)
	case "fs":
		log.Info("sourcing content from filesystem")
		return store.FromFS(cfg)
	}
}

func api(
	ctx context.Context,
	log *slog.Logger,
	cfg config.Config,
	cypher *token.Cypher,
	content store.Content,
) func() error {
	handler := routes(log, cfg, content, cypher)

	socket := config.OrDefault(
		cfg,
		"SOCKET",
		fmt.Sprintf(
			"%s:%s",
			config.OrDefault(cfg, "LISTEN_ADDRESS", "0.0.0.0"),
			config.OrDefault(cfg, "LISTEN_PORT", "8080"),
		),
	)

	return func() error {
		log := log.With(slog.String("service", "http"))
		var err error

		wg := &sync.WaitGroup{}
		srv := &http.Server{Addr: socket, Handler: handler}
		wg.Go(func() {
			<-ctx.Done()
			ctx, cancel := context.WithDeadline(
				context.Background(),
				time.Now().Add(15*time.Second),
			)
			defer cancel()

			localerr := srv.Shutdown(ctx)
			if errors.Is(localerr, http.ErrServerClosed) {
				return
			}
			err = localerr
		})

		log.Info("listening on socket", slog.String("socket", socket))
		err = srv.ListenAndServe()
		wg.Wait()
		return err
	}
}
