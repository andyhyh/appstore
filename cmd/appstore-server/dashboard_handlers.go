package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/pressly/chi"
	"github.com/uninett/appstore/pkg/search"
	"github.com/uninett/appstore/pkg/status"
	"html/template"
	helm_search "k8s.io/helm/cmd/helm/search"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/release"
	"net/http"
	"path"
	"path/filepath"
)

func ProcessTemplates(templatesDir string) map[string]*template.Template {
	layouts, err := filepath.Glob(path.Join(templatesDir, "layouts/*.html"))
	if err != nil {
		log.Fatal(err)
	}

	bases, err := filepath.Glob(path.Join(templatesDir, "bases/*.html"))
	if err != nil {
		log.Fatal(err)
	}

	processedTemplates := make(map[string]*template.Template)
	for _, layout := range layouts {
		files := append(bases, layout)
		tmpl := template.Must(template.ParseFiles(files...))
		processedTemplates[filepath.Base(layout)] = tmpl
	}

	return processedTemplates
}

func renderTemplate(w http.ResponseWriter, templates map[string]*template.Template, tmplName string, data interface{}) {
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

func makePackageIndexHandler(settings *helm_env.EnvSettings, templates map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results, err := search.GetAllCharts(settings)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		newestPackages := search.GetNewestVersion(results)
		search.SortByName(newestPackages)
		renderTemplate(w, templates, "index.html", struct {
			Results []*helm_search.Result
		}{newestPackages})
	}
}

func makePackageDetailHandler(settings *helm_env.EnvSettings, templates map[string]*template.Template) http.HandlerFunc {
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
		renderTemplate(w, templates, "package.html", struct {
			NewestVersion *helm_search.Result
			OtherVersions []*helm_search.Result
		}{newestVersion, otherVersions})
	}
}

func makeReleaseOverviewHandle(settings *helm_env.EnvSettings, templates map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := status.GetAllReleases(settings)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		renderTemplate(w, templates, "releases.html", struct{ Results []*release.Release }{res})
	}
}
