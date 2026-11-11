package filesystem

import (
	"fmt"
	"os"

	"github.com/go-git/go-billy/v5"
	"github.com/joaquinrovira/notes/internal/store/fsutil"
)

func New(name string) (Filesystem, error) {
	fs, err := os.OpenRoot(name)
	if err != nil {
		return Filesystem{}, fmt.Errorf("opening fs root: %w", err)
	}
	return Filesystem{Root: fs}, nil
}

type Filesystem struct {
	*os.Root
}

func (fs Filesystem) OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error) {
	f, err := fs.Root.OpenFile(filename, flag, perm)
	if err != nil {
		return nil, err
	}

	type file struct {
		*os.File
		fsutil.Locker
	}
	return &file{File: f}, err
}
