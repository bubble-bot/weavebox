package weavebox

import (
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"text/template"
)

// TemplateEngine provides simple, fast and powerfull rendering of HTML pages.
type TemplateEngine struct {
	root            string
	cache           map[string]*template.Template
	templates       []string
	templWithLayout map[string][]string
}

// NewTemplateEngine returns a new TemplateEngine object that will look for
// templates at the given root.
func NewTemplateEngine(root string) *TemplateEngine {
	return &TemplateEngine{
		cache:           map[string]*template.Template{},
		templWithLayout: map[string][]string{},
		root:            root,
	}
}

// Render renders the template and satisfies the weavebox.Renderer interface.
func (t *TemplateEngine) Render(w io.Writer, name string, data interface{}) error {
	if templ, exist := t.cache[name]; exist {
		return templ.ExecuteTemplate(w, "_", data)
	}
	return fmt.Errorf("template %s could not be found", name)
}

// SetTemplates sets single templates that not need to be parsed with a layout
func (t *TemplateEngine) SetTemplates(templates ...string) {
	for _, template := range templates {
		t.templates = append(t.templates, template)
	}
}

// SetTemplatesWithLayout sets a layout and parses all given templates with that
// layout.
// SetTemplatesWithLayout("layout.html",
// 		"user/index.html",
//		"user/list.html",
//		"user/create.html",
// )
func (t *TemplateEngine) SetTemplatesWithLayout(layout string, templates ...string) {
	t.templWithLayout[layout] = templates
}

// Init parses all the given singel and layout templates. And stores them in the
// template cache.
func (t *TemplateEngine) Init() {
	for layout, templates := range t.templWithLayout {
		layout, err := ioutil.ReadFile(path.Join(t.root, layout))
		handleErr(err)

		for _, page := range templates {
			parsedLayout, err := template.New("_").Parse(string(layout))
			handleErr(err)

			templ, err := ioutil.ReadFile(path.Join(t.root, page))
			handleErr(err)

			parsedTempl, err := parsedLayout.Parse(string(templ))
			handleErr(err)

			t.cache[page] = parsedTempl
		}
	}

	for _, file := range t.templates {
		templ, err := ioutil.ReadFile(path.Join(t.root, file))
		handleErr(err)

		parsedTempl, err := template.New("_").Parse(string(templ))
		handleErr(err)

		t.cache[file] = parsedTempl
	}
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}
