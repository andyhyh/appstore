package main

import (
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/pressly/chi"
	"github.com/pressly/chi/render"
	"github.com/uninett/appstore/pkg/dataporten"
	"github.com/uninett/appstore/pkg/helmutil"
	"github.com/uninett/appstore/pkg/install"
	"github.com/uninett/appstore/pkg/logger"
	"io"

	"github.com/uninett/appstore/pkg/search"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"net/http"
	"os"
)

type ErrorJson struct {
	Error string `json:"error"`
}

func returnJSON(w http.ResponseWriter, r *http.Request, res interface{}, err error, status int) {
	render.Status(r, status)
	if err != nil {
		render.JSON(w, r, ErrorJson{err.Error()})
	}
	render.JSON(w, r, res)
}

func chartSearchHandler(query string, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {
	results, err := search.FindCharts(settings, query, "", logger)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	return http.StatusOK, nil, results
}

func makeSearchForPackagesHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := logger.MakeAPILogger(r)
		query := chi.URLParam(r, "searchQuery")
		status, err, res := chartSearchHandler(query, settings, logger)

		returnJSON(w, r, res, err, status)
	}
}

func allPackagesHandler(settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {
	results, err := search.GetAllCharts(settings, logger)

	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	newestPackages := search.GetNewestVersion(results)
	search.SortByName(newestPackages)

	return http.StatusOK, nil, newestPackages
}

func makeListAllPackagesHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		status, err, res := allPackagesHandler(settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}

func installPackageHandler(packageName string, chartSettingsRaw io.ReadCloser, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {
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

	// TODO: Handle TLS related things:
	chartPath, err := install.LocateChartPath(packageName, chartSettings.Version, false, "", settings, logger)
	if err != nil {
		return http.StatusNotFound, err, nil
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

	res, err := install.InstallChart(chartPath, chartSettings, settings, logger)

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
		status, err, res := installPackageHandler(packageName, r.Body, settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}
