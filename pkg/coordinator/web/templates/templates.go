package templates

import (
	"bufio"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/tdewolff/minify"
)

var (
	//go:embed *
	templateFiles embed.FS
)

type Templates struct {
	logger   logrus.FieldLogger
	cache    map[string]*template.Template
	cacheMux sync.RWMutex
	funcs    template.FuncMap
	minify   bool
}

func New(logger logrus.FieldLogger, funcs template.FuncMap, minifyHTML bool) *Templates {
	return &Templates{
		logger: logger,
		cache:  make(map[string]*template.Template),
		funcs:  funcs,
		minify: minifyHTML,
	}
}

func (t *Templates) GetTemplate(files ...string) *template.Template {
	name := strings.Join(files, "-")

	t.cacheMux.RLock()
	if t.cache[name] != nil {
		defer t.cacheMux.RUnlock()
		return t.cache[name]
	}
	t.cacheMux.RUnlock()

	tmpl := template.New(name).Funcs(t.funcs)
	tmpl = template.Must(parseTemplateFiles(tmpl, t.readFileFS(templateFiles), files...))

	t.cacheMux.Lock()
	defer t.cacheMux.Unlock()

	t.cache[name] = tmpl

	return t.cache[name]
}

func (t *Templates) readFileFS(fsys fs.FS) func(string) (string, []byte, error) {
	return func(file string) (name string, b []byte, err error) {
		name = path.Base(file)
		b, err = fs.ReadFile(fsys, file)

		if t.minify {
			// minfiy template
			m := minify.New()
			m.AddFunc("text/html", t.minifyTemplate)

			b, err = m.Bytes("text/html", b)
			if err != nil {
				panic(err)
			}
		}

		return
	}
}

func (t *Templates) minifyTemplate(_ *minify.M, w io.Writer, r io.Reader, _ map[string]string) error {
	// remove newlines and spaces
	m1 := regexp.MustCompile(`([ \t]+)?[\r\n]+`)
	m2 := regexp.MustCompile(`([ \t])[ \t]+`)
	rb := bufio.NewReader(r)

	for {
		line, err := rb.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}

		line = m1.ReplaceAllString(line, "")
		line = m2.ReplaceAllString(line, " ")

		if _, errws := io.WriteString(w, line); errws != nil {
			return errws
		}

		if err == io.EOF {
			break
		}
	}

	return nil
}

func parseTemplateFiles(t *template.Template, readFile func(string) (string, []byte, error), filenames ...string) (*template.Template, error) {
	for _, filename := range filenames {
		name, b, err := readFile(filename)
		if err != nil {
			return nil, err
		}

		if t == nil {
			t = template.New(name)
		}

		var tmpl *template.Template
		if name == t.Name() {
			tmpl = t
		} else {
			tmpl = t.New(name)
		}

		_, err = tmpl.Parse(string(b))
		if err != nil {
			return nil, err
		}
	}

	return t, nil
}

func GetTemplateNames() []string {
	files, _ := getFileSysNames(templateFiles, ".")
	return files
}

func getFileSysNames(fsys fs.FS, dirname string) ([]string, error) {
	files := make([]string, 0, 100)

	entry, err := fs.ReadDir(fsys, dirname)
	if err != nil {
		return nil, fmt.Errorf("error reading embed directory, err: %w", err)
	}

	for _, f := range entry {
		info, err := f.Info()
		if err != nil {
			return nil, fmt.Errorf("error returning file info err: %w", err)
		}

		if !f.IsDir() {
			files = append(files, filepath.Join(dirname, info.Name()))
		} else {
			names, err := getFileSysNames(fsys, info.Name())
			if err != nil {
				return nil, err
			}

			files = append(files, names...)
		}
	}

	return files, nil
}
