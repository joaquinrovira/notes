package git

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

const remote = "origin"

func New(ctx context.Context, log *slog.Logger, url, user, token string) (billy.Filesystem, error) {
	dir, err := os.MkdirTemp("", "notes")
	if err != nil {
		return nil, fmt.Errorf("creating temporary directory: %w", err)
	}

	log.Debug("cloning into directory", slog.String("Path", dir))
	fs := osfs.New(dir)
	cache := cache.NewObjectLRUDefault()
	storer := filesystem.NewStorage(fs, cache)

	var auth transport.AuthMethod
	if user != "" && token != "" {
		auth = &githttp.BasicAuth{Username: user, Password: token}
	}
	repo, err := git.CloneContext(ctx, storer, fs, &git.CloneOptions{
		URL:          url,
		RemoteName:   remote,
		Depth:        1,
		SingleBranch: true,
		Auth:         auth,
	})
	if err != nil {
		return nil, fmt.Errorf("cloning remote %s: %w", url, err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	go updater(ctx, log, auth, repo, worktree)

	return fs, nil
}

func updater(
	ctx context.Context,
	log *slog.Logger,
	auth transport.AuthMethod,
	r *git.Repository,
	w *git.Worktree,
) {
	defer log.Debug("git updater exited")
	opts := &git.PullOptions{
		RemoteName:   remote,
		Depth:        1,
		SingleBranch: true,
		Auth:         auth,
	}
	for ctx.Err() == nil {
		err := w.PullContext(ctx, opts)
		switch {
		case errors.Is(err, git.NoErrAlreadyUpToDate):
		case err == nil:
			log.Info("updated content from remote")
			tryLogUpdateDetails(log, r)
		default:
			log.Warn("failed to retrieve from origin", slog.String("Err", err.Error()))
		}
		select {
		case <-ctx.Done():
		case <-time.After(15 * time.Second):
		}
	}
}

func tryLogUpdateDetails(
	log *slog.Logger,
	r *git.Repository,
) {
	err := func() error {
		ref, err := r.Head()
		if err != nil {
			return fmt.Errorf("retrieving HEAD: %w", err)
		}

		commit, err := r.CommitObject(ref.Hash())
		if err != nil {
			return fmt.Errorf("retrieving commit object: %w", err)
		}

		log.Info(
			"details for latest update",
			slog.String("Hash", commit.Hash.String()),
			slog.String("Author", fmt.Sprintf("%s <%s>", commit.Author.Name, commit.Author.Email)),
			slog.String("Message", commit.Message),
		)
		return nil
	}()
	if err != nil {
		log.Error("fetching update details", slog.String("Err", err.Error()))
	}
}
