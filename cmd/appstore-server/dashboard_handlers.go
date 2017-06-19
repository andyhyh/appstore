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
		results, err := search.GetAllCharts(settings)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

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
		if packageName == "" {
			http.Error(w, "Package not found!", http.StatusNotFound)
			return
		}

		packageVersions, err := search.GetSinglePackage(settings, packageName)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		newestVersion, otherVersions := packageVersions[0], packageVersions[1:len(packageVersions)]
		renderTemplate(w, templates, "package", struct {
			NewestVersion *helm_search.Result
			OtherVersions []*helm_search.Result
		}{newestVersion, otherVersions})
	}
}

func makeReleaseOverviewHandle(settings *helm_env.EnvSettings, templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := status.GetAllReleases(settings)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		renderTemplate(w, templates, "releases", struct{ Results []*release.Release }{res})
	}
}
