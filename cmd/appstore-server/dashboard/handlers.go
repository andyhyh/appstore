package dashboard

import (
	"html/template"
	"net/http"

	"github.com/go-chi/chi"

	"github.com/UNINETT/appstore/cmd/appstore-server/api"
	"github.com/UNINETT/appstore/pkg/logger"
	"github.com/UNINETT/appstore/pkg/templateutil"

	helm_search "k8s.io/helm/cmd/helm/search"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
)

func returnHTML(w http.ResponseWriter, template string, templates map[string]*template.Template, res interface{}, err error, status int) {
	if err != nil {
		http.Error(w, err.Error(), status)
	} else {
		templateutil.RenderTemplate(w, templates, template, res)
	}
}
func makePackageIndexHandler(settings *helm_env.EnvSettings, templates map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)

		status, err, newestPackages := api.AllPackagesHandler(settings, apiReqLogger)
		formattedRes := struct {
			Results [][]*helm_search.Result
		}{newestPackages}
		returnHTML(w, "index.html", templates, formattedRes, err, status)
	}
}

func makePackageDetailHandler(settings *helm_env.EnvSettings, templates map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		packageName := chi.URLParam(r, "packageName")
		version := chi.URLParam(r, "version")

		status, err, res := api.PackageDetailHandler(packageName, version, settings, apiReqLogger)
		if err != nil {
			status = http.StatusInternalServerError
		}
		var valuesRaw string
		var metadata *chart.Metadata
		if res != nil {
			valuesRaw = res.GetValues().GetRaw()
			metadata = res.GetMetadata()
		} else {
			status = http.StatusNotFound
		}

		formattedRes := struct {
			Package *chart.Metadata
			Values  string
		}{metadata, valuesRaw}
		returnHTML(w, "package.html", templates, formattedRes, err, status)
	}
}

func makeReleaseOverviewHandler(settings *helm_env.EnvSettings, templates map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		status, err, res := api.ReleaseOverviewHandler(settings, apiReqLogger)

		returnHTML(w, "releases.html", templates, struct{ Results []*release.Release }{res}, err, status)
	}
}
