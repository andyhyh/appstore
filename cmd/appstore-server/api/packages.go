package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi"

	"github.com/Sirupsen/logrus"

	"github.com/UNINETT/appstore/pkg/install"
	"github.com/UNINETT/appstore/pkg/logger"
	app_search "github.com/UNINETT/appstore/pkg/search"

	"k8s.io/helm/cmd/helm/search"
	"k8s.io/helm/pkg/chartutil"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

type PackageAppstoreMetaData struct {
	Repo string `json:"repo"`
}

const (
	defaultRepo = "stable"
)

// Show all information about a given package / chart
func PackageDetailHandler(packageName, repo, version string, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, *chart.Chart, error) {
	if packageName == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("no package specified")
	}

	if repo == "" {
		repo = defaultRepo
	}

	// TODO: Handle TLS related things:
	chartPath, err := install.LocateChartPath(packageName, repo, version, false, "", settings, logger)
	if err != nil {
		return http.StatusNotFound, nil, fmt.Errorf("%s, version: %s, repo: %s not found", packageName, version, repo)
	}

	chartRequested, err := chartutil.Load(chartPath)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return http.StatusOK, chartRequested, nil
}

func makePackageDetailHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)

		p := chi.URLParam(r, "packageName")
		v := r.URL.Query().Get("version")
		repo := r.URL.Query().Get("repo")

		status, res, err := PackageDetailHandler(p, repo, v, settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}

func groupSearchResult(results []*search.Result) []Package {
	packagesAllVersions := app_search.GroupResultsByName(results)
	packagesWithVersions := make([]Package, len(packagesAllVersions))
	for p_i, packages := range packagesAllVersions {
		versions := make([]string, len(packages))
		for v_i, pv := range packages {
			versions[v_i] = pv.Chart.Version
		}
		latestPackage := packages[0]
		parts := strings.Split(latestPackage.Name, "/")

		p := Package{packages[0], versions, parts[0]}
		packagesWithVersions[p_i] = p
	}

	return packagesWithVersions
}

// Find all chart matching a specific query, such as ?query=mysql or ?repo=stable.
func chartSearchHandler(query string, repo string, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, interface{}, error) {
	results, err := app_search.FindCharts(settings, query, repo, "", logger)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	packagesWithVersions := groupSearchResult(results)

	return http.StatusOK, packagesWithVersions, nil
}

type Package struct {
	NewestChart       *search.Result `json:"newest_chart"`
	AvailableVersions []string       `json:"available_versions"`
	Repo              string         `json:"repo"`
}

// Return a list of all packages paired with all available versions of the package.
func allPackagesHandler(settings *helm_env.EnvSettings, logger *logrus.Entry) (int, []Package, error) {
	results, err := app_search.GetAllCharts(settings, logger)

	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	packagesWithVersions := groupSearchResult(results)
	return http.StatusOK, packagesWithVersions, nil
}

func makeListPackagesHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		query := r.URL.Query().Get("query")
		repo := r.URL.Query().Get("repo")

		var status int
		var err error
		var res interface{}
		if query != "" || repo != "" {
			status, res, err = chartSearchHandler(query, repo, settings, apiReqLogger)
		} else {
			status, res, err = allPackagesHandler(settings, apiReqLogger)
		}

		returnJSON(w, r, res, err, status)
	}
}
