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

func (fs Filesystem) ReadDir(path string) ([]os.FileInfo, error) {
	f, err := fs.Root.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	entries, err := f.ReadDir(0)
	if err != nil {
		return nil, fmt.Errorf("file error: %w", err)
	}

	var infos []os.FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("reading file info %q: %w", entry.Name(), err)
		}
		infos = append(infos, info)
	}

	return infos, nil
}
