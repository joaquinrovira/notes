package git

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
)

const remote = "origin"

type Option func(*options)

func apply(v options, opts ...Option) options {
	for _, opt := range opts {
		safe(opt)(&v)
	}
	return v
}

func safe(o Option) Option {
	if o == nil {
		return func(o *options) {}
	}
	return o
}

func WithBasicAuth(user, pass string) Option {
	if user == "" {
		return nil
	}
	return func(o *options) {
		o.Auth = &githttp.BasicAuth{
			Username: user,
			Password: pass,
		}
	}
}

func WithBranch(name string) Option {
	return func(o *options) {
		o.Branch = name
	}
}

type options struct {
	Auth   transport.AuthMethod
	Branch string
}

func New(ctx context.Context, log *slog.Logger, wg *sync.WaitGroup, uri string, opts ...Option) (billy.Filesystem, error) {
	o := apply(options{}, opts...)
	auth := o.Auth

	dir, err := os.MkdirTemp("", "notes")
	if err != nil {
		return nil, fmt.Errorf("creating temporary directory: %w", err)
	}

	wg.Go(func() {
		<-ctx.Done()
		const msg = "cleaning up temporary clone directory"
		log.Debug(msg, slog.String("path", dir))
		err := os.RemoveAll(dir)
		if err != nil {
			log.Error(msg, slog.String("err", err.Error()))
		}
	})

	branch := o.Branch
	if branch == "" {
		b, err := queryRemoteDefaultBanch(ctx, uri, auth)
		if err != nil {
			return nil, fmt.Errorf("failed to discover default branch: %v", err)
		}
		branch = b
	}
	log.Debug("targeting remote branch", slog.String("name", branch))

	copts := &git.CloneOptions{
		URL:           uri,
		RemoteName:    remote,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
		Depth:         1,
		Tags:          git.NoTags,
		Auth:          auth,
	}

	local := func() bool {
		url, err := url.ParseRequestURI(uri)
		if err != nil {
			return true
		}
		switch url.Scheme {
		case "", "file":
			return true
		}
		return false
	}()
	if local {
		abs, err := filepath.Abs(uri)
		if err != nil {
			return nil, fmt.Errorf("resolving absolute path %s: %w", uri, err)
		}
		copts.URL = abs
		copts.Depth = 0
	}

	fs := osfs.New(dir)
	gitfs, err := fs.Chroot(".git")
	if err != nil {
		return nil, fmt.Errorf("opening .git in filesystem dir: %w", err)
	}
	cache := cache.NewObjectLRUDefault()
	storer := filesystem.NewStorage(gitfs, cache)

	log.Debug("cloning into directory", slog.String("path", dir))
	repo, err := git.CloneContext(ctx, storer, fs, copts)
	if err != nil {
		return nil, fmt.Errorf("cloning remote %s: %w", uri, err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	go updater(ctx, log, auth, repo, worktree, branch)

	return fs, nil
}

func queryRemoteDefaultBanch(ctx context.Context, uri string, auth transport.AuthMethod) (string, error) {
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: remote,
		URLs: []string{uri},
	})

	refs, err := remote.ListContext(ctx, &git.ListOptions{Auth: auth})
	if err != nil {
		return "", fmt.Errorf("listing remote: %w", err)
	}

	ref := func() *plumbing.Reference {
		for _, ref := range refs {
			if ref.Name() == "HEAD" && ref.Type() == plumbing.SymbolicReference {
				return ref
			}
		}
		return nil
	}()

	if ref == nil {
		return "", fmt.Errorf("missing ref HEAD")
	}

	if !ref.Target().IsBranch() {
		return "", fmt.Errorf("remote HEAD does not point to a default branch")
	}

	return ref.Target().Short(), nil
}

func updater(
	ctx context.Context,
	log *slog.Logger,
	auth transport.AuthMethod,
	r *git.Repository,
	w *git.Worktree,
	branch string,
) {
	defer log.Debug("updater exited")
	opts := &git.FetchOptions{
		RemoteName: remote,
		Depth:      1,
		Auth:       auth,
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/remotes/%s/%s", branch, remote, branch)),
		},
	}

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
		case <-time.After(15 * time.Second):
		}
		err := update(ctx, log, opts, r, w, branch)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Warn("update failure", slog.String("err", err.Error()))
		}
	}
}

func update(
	ctx context.Context,
	log *slog.Logger,
	opts *git.FetchOptions,
	r *git.Repository,
	w *git.Worktree,
	branch string,
) error {
	log.Debug("feching from remote")
	err := r.FetchContext(ctx, opts)
	if errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("feching from remote: %w", err)
	}

	target := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", remote, branch))
	ref, err := r.Reference(target, true)
	if err != nil {
		return fmt.Errorf("resolving remote reference: %w", err)
	}

	head := head(r)
	if head.Hash() == ref.Hash() {
		return nil
	}

	func() {
		log := log
		if v, err := loggerUpdateDetails(r, ref); err != nil {
			log.Error("fetching update details", slog.String("err", err.Error()))
		} else {
			log = slog.New(log.Handler().WithAttrs(v))
		}
		log.Info("fetched updates from remote")
	}()

	log.Debug("reseting worktree to latest commit")
	err = w.Reset(&git.ResetOptions{
		Mode:   git.HardReset,
		Commit: ref.Hash(),
	})
	if err != nil {
		return fmt.Errorf("reseting worktree to latest commit: %w", err)
	}

	return nil
}

func head(r *git.Repository) *plumbing.Reference {
	ref, err := r.Head()
	if err != nil {
		panic(err)
	}
	return ref
}

func loggerUpdateDetails(
	r *git.Repository,
	ref *plumbing.Reference,
) ([]slog.Attr, error) {
	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("retrieving commit object: %w", err)
	}

	message := commit.Message
	message = strings.TrimSpace(message)
	message = strings.Split(message, "\n")[0]
	message = strings.TrimSpace(message)

	return []slog.Attr{
		slog.String("hash", commit.Hash.String()[0:8]),
		slog.String("author", fmt.Sprintf("%s <%s>", commit.Author.Name, commit.Author.Email)),
		slog.String("message", message),
	}, nil
}
