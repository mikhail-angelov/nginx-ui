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

// Template manages HTML templates loaded from embedded files.
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

// Render executes and renders the named template with the given data.
func (t *Template) Render(w http.ResponseWriter, name string, data any) error {
	tmpl, ok := t.templates[name]
	if !ok {
		http.Error(w, fmt.Sprintf("The template %s does not exist.", name),
			http.StatusInternalServerError)
		return fmt.Errorf("template %s does not exist", name)
	}

	e := tmpl.ExecuteTemplate(w, name, data)
	if e != nil {
		log.Printf("Failed to render template %s: %v", name, e)
	}
	return fmt.Errorf("failed to execute template: %w", e)
}

// SubRender executes and renders a specific component within the named template.
func (t *Template) SubRender(w http.ResponseWriter, name, component string, data any) error {
	tmpl, ok := t.templates[name]
	if !ok {
		http.Error(w, fmt.Sprintf("The template %s does not exist.", name),
			http.StatusInternalServerError)
		return fmt.Errorf("template %s does not exist", name)
	}
	e := tmpl.ExecuteTemplate(w, component, data)
	if e != nil {
		log.Printf("Failed to render component %s in template %s: %v", component, name, e)
	}
	return fmt.Errorf("failed to execute template component: %w", e)
}

func (t *Template) loadTemplates(embedFs embed.FS) {
	layoutFiles, err := fs.Glob(embedFs, "ui/templates/components/*.html")
	if err != nil {
		log.Panicf("failed to load components templates: %v", err)
	}
	includeFiles, err := fs.Glob(embedFs, "ui/templates/pages/*.html")
	if err != nil {
		log.Panicf("failed to load page templates: %v", err)
	}

	for _, file := range includeFiles {
		files := make([]string, len(layoutFiles))
		copy(files, layoutFiles)
		files = append(files, file)
		tmpl := template.Must(template.ParseFS(embedFs, files...))
		name := strings.TrimSuffix(filepath.Base(file), ".html")
		t.templates[name] = tmpl
	}
	log.Println("templates loading successful")
}
