package builtin

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
)

//go:embed all:data
var content embed.FS
var mem billy.Filesystem = memfs.New()

type Filesystem struct{}

func (f *Filesystem) OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error) {
	_, err := mem.Stat(filename)
	if err == nil {
		return mem.Open(filename)
	}

	file, err := content.Open(path.Join("data/", filename))
	if err != nil {
		return nil, fmt.Errorf("opening builtin '%s': %w", filename, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading builtin '%s': %w", filename, err)
	}

	memfile, err := mem.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("creating builtin '%s': %w", filename, err)
	}

	_, err = memfile.Write(data)
	if err != nil {
		return nil, fmt.Errorf("writing builtin '%s': %w", filename, err)
	}

	memfile.Seek(0, io.SeekStart)
	return memfile, nil
}

func (f *Filesystem) Stat(filename string) (os.FileInfo, error) {
	info, err := mem.Stat(filename)
	if err == nil {
		return info, nil
	}

	file, err := f.OpenFile(filename, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("reading builtin '%s': %w", filename, err)
	}

	err = file.Close()
	if err != nil {
		return nil, fmt.Errorf("closgin builtin '%s': %w", filename, err)
	}

	return mem.Stat(filename)
}
