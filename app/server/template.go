package server

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

type Template struct {
	templates map[string]*template.Template
}

// NewTemplate creates a new Template instance
func NewTemplate(embedFs embed.FS) *Template {
	t := &Template{
		templates: make(map[string]*template.Template),
	}
	t.loadTemplates(embedFs)
	return t
}

func (t *Template) Render(w http.ResponseWriter, name string, data interface{}) error {
	tmpl, ok := t.templates[name]
	if !ok {
		http.Error(w, fmt.Sprintf("The template %s does not exist.", name),
			http.StatusInternalServerError)
	}

	e := tmpl.ExecuteTemplate(w, name, data)
	if e != nil {
		print(e)
	}
	return e
}
func (t *Template) SubRender(w http.ResponseWriter, name string, component string, data interface{}) error {
	tmpl, ok := t.templates[name]
	if !ok {
		http.Error(w, fmt.Sprintf("The template %s does not exist.", name),
			http.StatusInternalServerError)
	}
	e := tmpl.ExecuteTemplate(w, component, data)
	if e != nil {
		print(e)
	}
	return e
}

func (t *Template) loadTemplates(embedFs embed.FS) {
	layoutFiles, err := fs.Glob(embedFs, "ui/templates/components/*.html")
	if err != nil {
		log.Fatalf("failed to load components templates: %v", err)
	}
	includeFiles, err := fs.Glob(embedFs, "ui/templates/pages/*.html")
	if err != nil {
		log.Fatalf("failed to load page templates: %v", err)
	}

	for _, file := range includeFiles {
		files := append(layoutFiles, file)
		tmpl := template.Must(template.ParseFS(embedFs, files...))
		name := strings.TrimSuffix(filepath.Base(file), ".html")
		t.templates[name] = tmpl
	}
	log.Println("templates loading successful")
}
