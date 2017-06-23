package templateutil

import (
	"html/template"
	"net/http"
	"path"
	"path/filepath"
)

func ProcessTemplates(templatesDir string) (map[string]*template.Template, error) {
	layouts, err := filepath.Glob(path.Join(templatesDir, "layouts/*.html"))
	if err != nil {
		return nil, err
	}

	bases, err := filepath.Glob(path.Join(templatesDir, "bases/*.html"))
	if err != nil {
		return nil, err
	}

	processedTemplates := make(map[string]*template.Template)
	for _, layout := range layouts {
		files := append(bases, layout)
		tmpl := template.Must(template.ParseFiles(files...))
		processedTemplates[filepath.Base(layout)] = tmpl
	}

	return processedTemplates, nil
}

func RenderTemplate(w http.ResponseWriter, templates map[string]*template.Template, tmplName string, data interface{}) {
	tmpl, found := templates[tmplName]

	if !found {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}

	err := tmpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
