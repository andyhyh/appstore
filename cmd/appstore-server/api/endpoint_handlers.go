package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/pressly/chi"
	"github.com/pressly/chi/render"

	"github.com/uninett/appstore/pkg/dataporten"
	"github.com/uninett/appstore/pkg/helmutil"
	"github.com/uninett/appstore/pkg/install"
	"github.com/uninett/appstore/pkg/logger"
	"github.com/uninett/appstore/pkg/releaseutil"
	app_search "github.com/uninett/appstore/pkg/search"
	"github.com/uninett/appstore/pkg/status"

	"k8s.io/helm/cmd/helm/search"
	"k8s.io/helm/pkg/chartutil"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
)

type ErrorJson struct {
	Error string `json:"error"`
}

func returnJSON(w http.ResponseWriter, r *http.Request, res interface{}, err error, status int) {
	render.Status(r, status)
	if err != nil {
		render.JSON(w, r, ErrorJson{err.Error()})
	} else {
		render.JSON(w, r, res)
	}
}

func PackageDetailHandler(packageName string, version string, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, *chart.Chart) {
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

	return http.StatusOK, nil, chartRequested
}

func chartSearchHandler(query string, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {
	results, err := app_search.FindCharts(settings, query, "", logger)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	return http.StatusOK, nil, results
}

func AllPackagesHandler(settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, [][]*search.Result) {
	results, err := app_search.GetAllCharts(settings, logger)

	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	packagesAllVersions := app_search.GroupResultsByName(results)

	return http.StatusOK, nil, packagesAllVersions
}

func makeListAllPackagesHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		query := r.URL.Query().Get("query")
		if query == "" {
			status, err, res := AllPackagesHandler(settings, apiReqLogger)
			returnJSON(w, r, res, err, status)
		} else {
			status, err, res := chartSearchHandler(query, settings, apiReqLogger)
			returnJSON(w, r, res, err, status)
		}
	}
}

func releaseStatusHandler(releaseName string, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {
	if releaseName == "" {
		return http.StatusNotFound, fmt.Errorf("no release provided"), nil
	}
	client := helmutil.InitHelmClient(settings)
	status, err := client.ReleaseStatus(releaseName)
	logger.Debugf("Attemping to fetch the status of: %s", releaseName)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	return http.StatusOK, err, status
}

func makeReleaseStatusHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		releaseName := chi.URLParam(r, "releaseName")
		status, err, res := releaseStatusHandler(releaseName, settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}

func installPackageHandler(releaseSettingsRaw io.ReadCloser, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {
	var releaseSettings *releaseutil.ReleaseSettings
	decoder := json.NewDecoder(releaseSettingsRaw)
	err := decoder.Decode(&releaseSettings)

	if err != nil {
		logger.Debugf("Error decoding the POSTed JSON: '%s, %s'", releaseSettingsRaw, err.Error())
		return http.StatusBadRequest, fmt.Errorf("invalid json"), nil
	}

	status, err, chartRequested := PackageDetailHandler(releaseSettings.Package, releaseSettings.Version, settings, logger)
	if status != http.StatusOK {
		return status, err, nil
	}

	dataportenSettings, err := dataporten.MaybeGetSettings(releaseSettings.Values)

	if dataportenSettings != nil && err == nil {
		logger.Debugf("Attempting to register dataporten application %s", dataportenSettings.Name)
		regResp, err := dataporten.CreateClient(dataportenSettings, os.Getenv("TOKEN"), logger)

		if regResp.StatusCode != http.StatusCreated {
			return regResp.StatusCode, fmt.Errorf(regResp.Status), nil
		}

		_, err = dataporten.ParseRegistrationResult(regResp.Body, logger)
		if err != nil {
			return http.StatusInternalServerError, err, nil
		}
		logger.Debugf("Successfully registered application %s", dataportenSettings.Name)
	}

	res, err := install.InstallChart(chartRequested, releaseSettings.Values, settings, logger)

	if err == nil {
		return http.StatusOK, nil, releaseutil.Release{Id: res.Name, Owner: "", ReleaseSettings: releaseSettings}
	} else {
		return http.StatusInternalServerError, err, nil
	}
}

func makeInstallPackageHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		status, err, res := installPackageHandler(r.Body, settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}

func ReleaseOverviewHandler(settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, []*release.Release) {
	res, err := status.GetAllReleases(settings, logger)

	if err != nil {
		return http.StatusInternalServerError, err, nil
	}
	return http.StatusOK, nil, res
}

func makeReleaseOverviewHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		status, err, res := ReleaseOverviewHandler(settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}
