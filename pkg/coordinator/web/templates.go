package web

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/tdewolff/minify"

	"github.com/ethpandaops/assertoor/pkg/coordinator/web/utils"
)

var templateFiles fs.FS
var templateCache = make(map[string]*template.Template)
var templateCacheMux = &sync.RWMutex{}
var templateFuncs = utils.GetTemplateFuncs()

func GetTemplate(files ...string) *template.Template {
	name := strings.Join(files, "-")

	templateCacheMux.RLock()
	if templateCache[name] != nil {
		defer templateCacheMux.RUnlock()
		return templateCache[name]
	}
	templateCacheMux.RUnlock()

	tmpl := template.New(name).Funcs(templateFuncs)
	tmpl = template.Must(parseTemplateFiles(tmpl, readFileFS(templateFiles), files...))

	templateCacheMux.Lock()
	defer templateCacheMux.Unlock()

	templateCache[name] = tmpl

	return templateCache[name]
}

func readFileFS(fsys fs.FS) func(string) (string, []byte, error) {
	return func(file string) (name string, b []byte, err error) {
		name = path.Base(file)
		b, err = fs.ReadFile(fsys, file)

		if frontendConfig.Minify {
			// minfiy template
			m := minify.New()
			m.AddFunc("text/html", minifyTemplate)

			b, err = m.Bytes("text/html", b)
			if err != nil {
				panic(err)
			}
		}

		return
	}
}

func minifyTemplate(_ *minify.M, w io.Writer, r io.Reader, _ map[string]string) error {
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

func CompileTimeCheck(fsys fs.FS) error {
	files, err := getFileSysNames(fsys, ".")
	if err != nil {
		return err
	}

	template.Must(template.New("layout").Funcs(templateFuncs).ParseFS(templateFiles, files...))
	logger.Infof("compile time check completed")

	return nil
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
