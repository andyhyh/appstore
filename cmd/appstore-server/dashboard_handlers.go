package main

import (
	"github.com/uninett/appstore/pkg/search"
	"html/template"
	helm_search "k8s.io/helm/cmd/helm/search"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"net/http"
)

func renderTemplate(w http.ResponseWriter, templates *template.Template, tmpl_name string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl_name+".html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func makePackageIndexHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		templates := template.Must(template.ParseFiles("ui/templates/index.html"))
		res, _ := search.GetAllCharts(settings)
		renderTemplate(w, templates, "index", struct{ Results []*helm_search.Result }{res})
	}
}
