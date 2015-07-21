package weavebox

import (
	"fmt"
	"io"
	"io/ioutil"
	"text/template"
)

type TemplateEngine struct {
	cache           map[string]*template.Template
	layout          string
	templates       []string
	templWithLayout map[string][]string
}

func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{
		cache:           map[string]*template.Template{},
		templWithLayout: map[string][]string{},
	}
}

func (t *TemplateEngine) Render(w io.Writer, name string, data interface{}) error {
	if templ, exist := t.cache[name]; exist {
		return templ.ExecuteTemplate(w, "_", data)
	}
	return fmt.Errorf("template %s could not be found", name)
}

func (t *TemplateEngine) SetLayout(s string) {
	t.layout = s
}

func (t *TemplateEngine) SetTemplates(templates ...string) {
	for _, template := range templates {
		t.templates = append(t.templates, template)
	}
}

func (t *TemplateEngine) SetTemplatesWithLayout(layout string, templates ...string) {
	t.templWithLayout[layout] = templates
}

func (t *TemplateEngine) Init() {
	for layout, templates := range t.templWithLayout {
		layout, err := ioutil.ReadFile(layout)
		handleErr(err)

		for _, page := range templates {
			parsedLayout, err := template.New("_").Parse(string(layout))
			handleErr(err)

			templ, err := ioutil.ReadFile(page)
			handleErr(err)

			parsedTempl, err := parsedLayout.Parse(string(templ))
			handleErr(err)

			t.cache[page] = parsedTempl
		}
	}

	for _, file := range t.templates {
		templ, err := ioutil.ReadFile(file)
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
