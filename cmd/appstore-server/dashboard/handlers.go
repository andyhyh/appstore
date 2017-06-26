package dashboard

import (
	"html/template"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/pressly/chi"

	"github.com/uninett/appstore/cmd/appstore-server/api"
	//"github.com/uninett/appstore/pkg/install"
	"github.com/uninett/appstore/pkg/logger"
	"github.com/uninett/appstore/pkg/status"
	"github.com/uninett/appstore/pkg/templateutil"

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
		}

		formattedRes := struct {
			Package *chart.Metadata
			Values  string
		}{metadata, valuesRaw}
		returnHTML(w, "package.html", templates, formattedRes, err, status)
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
