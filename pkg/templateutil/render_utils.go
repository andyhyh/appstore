package templateutil

import (
	"fmt"
	"html/template"
	"net/http"
	"path"
	"path/filepath"
)

func isMap(item interface{}) bool {
	switch item.(type) {
	case map[string]interface{}:
		return true
	default:
		return false
	}
}

func getType(item interface{}) string {
	switch v := item.(type) {
	case string:
		return "string"
	case bool:
		return "bool"
	case int, float64, float32:
		return "int"
	case uint:
		return "uint"
	case []interface{}:
		return "list"
	default:
		return fmt.Sprintf("%T", v)
	}
}

func ProcessTemplates(templatesDir string) (map[string]*template.Template, error) {
	layouts, err := filepath.Glob(path.Join(templatesDir, "layouts/*.html"))
	if err != nil {
		return nil, err
	}

	bases, err := filepath.Glob(path.Join(templatesDir, "bases/*.html"))
	if err != nil {
		return nil, err
	}

	funcMap := template.FuncMap{"isMap": isMap, "getType": getType}

	processedTemplates := make(map[string]*template.Template)
	for _, layout := range layouts {
		files := append(bases, layout)
		tmpl := template.Must(template.New(layout).Funcs(funcMap).ParseFiles(files...))
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
