package filesystem

import (
	"fmt"
	"os"

	"github.com/go-git/go-billy/v5"
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
	return &lockable{File: f}, err
}

// Naive implementation Lock and Unlock of [billy.File] for [*os.File].
// Just have to satisfy the interface. We're not really using locks.
type lockable struct {
	*os.File
}

func (f *lockable) Lock() error {
	return nil
}
func (f *lockable) Unlock() error {
	return nil
}
