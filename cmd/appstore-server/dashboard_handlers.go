package main

import (
	"github.com/pressly/chi"
	"github.com/uninett/appstore/pkg/search"
	"github.com/uninett/appstore/pkg/status"
	"html/template"
	helm_search "k8s.io/helm/cmd/helm/search"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/release"
	"net/http"
)

func renderTemplate(w http.ResponseWriter, templates *template.Template, tmpl_name string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl_name+".html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func makePackageIndexHandler(settings *helm_env.EnvSettings, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results, _ := search.GetAllCharts(settings)
		newestPackages := search.GetNewestVersion(results)
		search.SortByName(newestPackages)
		renderTemplate(w, templates, "index", struct {
			Results []*helm_search.Result
		}{newestPackages})
	}
}

func makePackageDetailHandler(settings *helm_env.EnvSettings, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		packageName := chi.URLParam(req, "packageName")
		res, _ := search.GetSinglePackage(settings, packageName)
		renderTemplate(w, templates, "package", struct{ Package *helm_search.Result }{res})
	}
}

func makeReleaseOverviewHandle(settings *helm_env.EnvSettings, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, _ := status.GetAllReleases(settings)
		renderTemplate(w, templates, "releases", struct{ Results []*release.Release }{res})
	}
}
