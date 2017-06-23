package main

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

	app_search "github.com/uninett/appstore/pkg/search"
	"k8s.io/helm/cmd/helm/search"
	helm_env "k8s.io/helm/pkg/helm/environment"
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

func chartSearchHandler(query string, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {
	results, err := app_search.FindCharts(settings, query, "", logger)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	return http.StatusOK, nil, results
}

func packageUserValuesHandler(packageName string, version string, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {

	status, err, res := packageDetailHandler(packageName, version, settings, logger)

	if status != http.StatusOK {
		return status, err, nil
	}

	userVals, err := install.GetValsByKey("persistence", res.GetValues().GetRaw(), logger)
	return http.StatusOK, err, userVals
}

func makePackageUserValuesHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := logger.MakeAPILogger(r)
		packageName := chi.URLParam(r, "packageName")
		packageVersion := chi.URLParam(r, "version")

		status, err, res := packageUserValuesHandler(packageName, packageVersion, settings, logger)
		jsonString, err := json.Marshal(res)

		if err != nil {
			logger.Warn(err)
		} else {
			logger.Infof("%s", jsonString)
		}

		returnJSON(w, r, res, err, status)
	}
}

func makeSearchForPackagesHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := logger.MakeAPILogger(r)
		query := chi.URLParam(r, "searchQuery")
		status, err, res := chartSearchHandler(query, settings, logger)

		returnJSON(w, r, res, err, status)
	}
}

func allPackagesHandler(settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, [][]*search.Result) {
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
		status, err, res := allPackagesHandler(settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}

func installPackageHandler(packageName string, version string, chartSettingsRaw io.ReadCloser, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {
	if packageName == "" {
		return http.StatusNotFound, fmt.Errorf("package not found"), nil
	}
	logger.Debug("Attempting to install package: " + packageName)

	chartSettings := new(helmutil.ChartSettings)
	decoder := json.NewDecoder(chartSettingsRaw)
	err := decoder.Decode(&chartSettings)

	if err != nil {
		logger.Debugf("Error decoding the POSTed JSON: '%s, %s'", chartSettingsRaw, err.Error())
		return http.StatusBadRequest, fmt.Errorf("invalid json"), nil
	}

	status, err, chartRequested := packageDetailHandler(packageName, version, settings, logger)
	if status != http.StatusOK {
		return status, err, nil
	}

	if chartSettings.DataportenClientSettings.Name != "" {
		regResp, err := dataporten.CreateClient(chartSettings.DataportenClientSettings, os.Getenv("TOKEN"), logger)

		if regResp.StatusCode == http.StatusBadRequest {
			return http.StatusBadRequest, fmt.Errorf(regResp.Status), nil
		}

		regRes, err := dataporten.ParseRegistrationResult(regResp.Body, logger)
		if err != nil {
			return http.StatusInternalServerError, err, nil
		}
		logger.Debug(regRes)
	}

	res, err := install.InstallChart(chartRequested, chartSettings, settings, logger)

	if err == nil {
		return http.StatusOK, nil, res
	} else {
		return http.StatusInternalServerError, err, nil
	}
}

func makeInstallPackageHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		packageName := chi.URLParam(r, "packageName")
		version := chi.URLParam(r, "version")
		status, err, res := installPackageHandler(packageName, version, r.Body, settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}
