package server

import (
	"html/template"
	"net/http"
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w http.ResponseWriter, name string, data interface{}) error {
	e := t.templates.ExecuteTemplate(w, name, data)
	if e != nil {
		print(e)
	}
	return e
}

func LoadTemplates(path string) *Template {
	tmpl := template.Must(template.ParseGlob(path))
	t := &Template{
		templates: tmpl,
	}
	return t
}
