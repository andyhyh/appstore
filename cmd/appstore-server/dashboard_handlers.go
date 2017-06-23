package main

import (
	"fmt"
	"html/template"
	"net/http"
	"path"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/pressly/chi"

	"github.com/uninett/appstore/pkg/install"
	"github.com/uninett/appstore/pkg/logger"
	"github.com/uninett/appstore/pkg/status"

	helm_search "k8s.io/helm/cmd/helm/search"
	"k8s.io/helm/pkg/chartutil"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
)

func returnHTML(w http.ResponseWriter, template string, templates map[string]*template.Template, res interface{}, err error, status int) {
	if err != nil {
		http.Error(w, err.Error(), status)
	} else {
		renderTemplate(w, templates, template, res)
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

	processedTemplates := make(map[string]*template.Template)
	for _, layout := range layouts {
		files := append(bases, layout)
		tmpl := template.Must(template.ParseFiles(files...))
		processedTemplates[filepath.Base(layout)] = tmpl
	}

	return processedTemplates, nil
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
		apiReqLogger := logger.MakeAPILogger(r)

		status, err, newestPackages := allPackagesHandler(settings, apiReqLogger)
		formattedRes := struct {
			Results []*helm_search.Result
		}{newestPackages}
		returnHTML(w, "index.html", templates, formattedRes, err, status)
	}
}

func packageDetailHandler(packageName string, version string, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {
	if packageName == "" {
		return http.StatusBadRequest, fmt.Errorf("no package specified"), nil
	}

	// TODO: Handle TLS related things:
	chartPath, err := install.LocateChartPath(packageName, version, false, "", settings, logger)
	if err != nil {
		return http.StatusNotFound, err, nil
	}

	chartRequested, err := chartutil.Load(chartPath)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	return http.StatusOK, nil, struct {
		Result *chart.Chart
	}{chartRequested}
}

func makePackageDetailHandler(settings *helm_env.EnvSettings, templates map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		packageName := chi.URLParam(r, "packageName")
		version := chi.URLParam(r, "version")

		status, err, res := packageDetailHandler(packageName, version, settings, apiReqLogger)

		returnHTML(w, "package.html", templates, res, err, status)
	}
}

func releaseOverviewHandler(settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, []*release.Release) {
	res, err := status.GetAllReleases(settings, logger)

	if err != nil {
		return http.StatusInternalServerError, err, nil
	}
	return http.StatusOK, nil, res
}

func makeReleaseOverviewHandle(settings *helm_env.EnvSettings, templates map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		status, err, res := releaseOverviewHandler(settings, apiReqLogger)

		returnHTML(w, "releases.html", templates, struct{ Results []*release.Release }{res}, err, status)
	}
}
