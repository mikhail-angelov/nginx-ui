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

func LoadTemplates(embedFs embed.FS) *Template {
	templates := make(map[string]*template.Template)
	layoutFiles, err := fs.Glob(embedFs, "ui/templates/components/*.html")
	if err != nil {
		log.Fatal(err)
	}

	includeFiles, err := fs.Glob(embedFs, "ui/templates/pages/*.html")
	if err != nil {
		log.Fatal(err)
	}

	mainTemplate := template.New("main")

	if err != nil {
		log.Fatal(err)
	}
	for _, file := range includeFiles {
		fileName := strings.TrimSuffix(filepath.Base(file), ".html")
		files := append(layoutFiles, file)
		templates[fileName], err = mainTemplate.Clone()
		if err != nil {
			log.Fatal(err)
		}
		templates[fileName] = template.Must(templates[fileName].ParseFiles(files...))
	}

	log.Println("templates loading successful")
	t := &Template{
		templates: templates,
	}
	return t
}
