package union

import (
	"errors"
	"os"

	"github.com/go-git/go-billy/v5"
)

// ErrNoFilesystems is returned when no sub-filesystems are provided.
var ErrNoFilesystems = errors.New("no sub-filesystems configured")

type Filesystem interface {
	OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error)
	Stat(filename string) (os.FileInfo, error)
}

// Multi redirects operations across sub-filesystems in order, returning the
// first successful result or the last error encountered.
type Multi struct {
	filesystems []Filesystem
}

// New creates a Multi filesystem from one or more sub-filesystems.
func New(filesystems ...Filesystem) *Multi {
	return &Multi{
		filesystems: filesystems,
	}
}

func (m *Multi) OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error) {
	if len(m.filesystems) == 0 {
		return nil, ErrNoFilesystems
	}

	var lastErr error
	for _, fs := range m.filesystems {
		f, err := fs.OpenFile(filename, flag, perm)
		if err == nil {
			return f, nil
		}
		lastErr = err
	}

	return nil, lastErr
}

func (m *Multi) Stat(filename string) (os.FileInfo, error) {
	if len(m.filesystems) == 0 {
		return nil, ErrNoFilesystems
	}

	var lastErr error
	for _, fs := range m.filesystems {
		info, err := fs.Stat(filename)
		if err == nil {
			return info, nil
		}
		lastErr = err
	}

	return nil, lastErr
}
