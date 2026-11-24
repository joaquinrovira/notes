package countdown

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/joaquinrovira/notes/internal/store"
	"github.com/joaquinrovira/notes/internal/util/fp"
)

func CountdownInDir(fs store.Filesystem, dir string) (t *template.Template, err error) {
	location := path.Join(".", dir)
	if location == "" {
		return nil, fmt.Errorf("empty directory string")
	}

	stat, err := fs.Stat(location)
	if err != nil {
		return nil, fmt.Errorf("could not stat %q: %w", location, err)
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("not a directory %q", location)
	}

	files, err := fs.ReadDir(location)
	if err != nil {
		return nil, fmt.Errorf("could not read dir %q: %w", location, err)
	}

	if len(files) < 1 {
		return nil, fmt.Errorf("empty dir %q: %w", location, err)
	}

	names := fp.Filter(fp.Map(fp.Filter(
		files,
		func(v os.FileInfo) bool { return !v.IsDir() }),
		func(v os.FileInfo) string { return v.Name() }),
		func(v string) bool { return strings.HasPrefix(v, "countdown.html") })

	if len(names) < 1 {
		return nil, fmt.Errorf("no matches in dir %q: %w", location, err)
	}

	name := names[0]
	location = path.Join(location, name)
	stat, err = fs.Stat(location)
	if err != nil {
		return nil, fmt.Errorf("could not stat %q: %w", location, err)
	}

	file, err := fs.OpenFile(location, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("opening %q: %w", location, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", location, err)
	}

	tmpl, err := template.New("countdown-" + location).Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("invalid template %q: %w", location, err)
	}

	return tmpl, nil
}
