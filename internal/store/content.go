package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/go-git/go-billy/v5"
	"github.com/joaquinrovira/notes/internal/config"
	"github.com/joaquinrovira/notes/internal/store/filesystem"
	"github.com/joaquinrovira/notes/internal/store/git"
)

type Content interface {
	Data() Filesystem
	Static() Filesystem
}

type Filesystem interface {
	OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error)
	ReadDir(path string) ([]os.FileInfo, error)
	Stat(filename string) (os.FileInfo, error)
}

func FromGit(ctx context.Context, log *slog.Logger, cfg config.Config, wg *sync.WaitGroup) (Content, error) {
	var acc []error
	url, err := config.Require(cfg, "GIT_URL")
	acc = append(acc, err)
	user := config.OrDefault(cfg, "GIT_USER", "git")
	token, _ := cfg.String("GIT_TOKEN")
	// acc = append(acc, err)
	if err := errors.Join(acc...); err != nil {
		return nil, err
	}

	fs, err := git.New(ctx, log, wg, url, git.WithBasicAuth(user, token))
	if err != nil {
		return nil, fmt.Errorf("git content: %w", err)
	}

	return NewContent(
		prefixed("data", fs),
		prefixed("static", fs),
	), nil
}

func FromFS(cfg config.Config) (Content, error) {
	var acc []error

	dir := "."
	if path, ok := cfg.String("STORE_FS_PATH"); ok {
		dir = path
	}

	data, err := filesystem.New(filepath.Join(dir, "data"))
	acc = append(acc, decorate(err, func(err error) error {
		return fmt.Errorf("creating data fs store: %w", err)
	}))
	static, err := filesystem.New(filepath.Join(dir, "static"))
	acc = append(acc, decorate(err, func(err error) error {
		return fmt.Errorf("creating static fs store: %w", err)
	}))
	if err := errors.Join(acc...); err != nil {
		return nil, err
	}
	return NewContent(data, static), nil
}

func decorate(err error, fn func(error) error) error {
	if err == nil {
		return nil
	}
	return fn(err)
}

func NewContent(data, static Filesystem) Content {
	return contentfn{data: data, static: static}
}

type contentfn struct {
	data   Filesystem
	static Filesystem
}

func (c contentfn) Data() Filesystem   { return c.data }
func (c contentfn) Static() Filesystem { return c.static }

func prefixed(s string, fs Filesystem) Filesystem {
	return filesystemfn{
		openFile: func(filename string, flag int, perm os.FileMode) (billy.File, error) {
			return fs.OpenFile(path.Join(s, filename), flag, perm)
		},
		stat: func(filename string) (os.FileInfo, error) {
			return fs.Stat(path.Join(s, filename))
		},
		readDir: func(dir string) ([]os.FileInfo, error) {
			return fs.ReadDir(path.Join(s, dir))
		},
	}
}

type filesystemfn struct {
	openFile func(filename string, flag int, perm os.FileMode) (billy.File, error)
	stat     func(filename string) (os.FileInfo, error)
	readDir  func(path string) ([]os.FileInfo, error)
}

func (f filesystemfn) OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error) {
	return f.openFile(filename, flag, perm)
}

func (f filesystemfn) Stat(filename string) (os.FileInfo, error) {
	return f.stat(filename)
}

func (f filesystemfn) ReadDir(path string) ([]os.FileInfo, error) {
	return f.readDir(path)
}
