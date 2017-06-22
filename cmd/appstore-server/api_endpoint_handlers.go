package main

import (
	"encoding/json"
	"github.com/pressly/chi"
	"github.com/pressly/chi/render"
	"github.com/uninett/appstore/pkg/dataporten"
	"github.com/uninett/appstore/pkg/helmutil"
	"github.com/uninett/appstore/pkg/install"
	"github.com/uninett/appstore/pkg/logger"
	"github.com/uninett/appstore/pkg/search"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"net/http"
	"os"
)

func writeErrorJson(w http.ResponseWriter, r *http.Request, err error, status int) {
	render.Status(r, status)
	render.JSON(w, r, struct{ Error string }{err.Error()})
}

func makeSearchForPackagesHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := logger.MakeAPILogger(r)
		query := chi.URLParam(r, "searchQuery")
		results, err := search.FindCharts(settings, query, "", logger)

		if err != nil {
			writeErrorJson(w, r, err, http.StatusInternalServerError)
			return
		}

		render.JSON(w, r, results)
	}
}

func makeListAllPackagesHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		results, err := search.GetAllCharts(settings, apiReqLogger)

		if err != nil {
			writeErrorJson(w, r, err, http.StatusInternalServerError)
			return
		}

		newestPackages := search.GetNewestVersion(results)
		search.SortByName(newestPackages)
		render.JSON(w, r, (newestPackages))
	}
}

func makeInstallPackageHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		packageName := chi.URLParam(r, "packageName")
		if packageName == "" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		apiReqLogger.Debug("Installing package: " + packageName)

		chartSettings := new(helmutil.ChartSettings)
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&chartSettings)

		if err != nil {
			apiReqLogger.Debugf("Error decoding the POSTed JSON: '%s, %s'", r.Body, err.Error())
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, "Invalid JSON!")
			return
		}

		// TODO: Handle TLS related things:
		chartPath, err := install.LocateChartPath(packageName, chartSettings.Version, false, "", settings, apiReqLogger)
		if err != nil {
			writeErrorJson(w, r, err, http.StatusNotFound)
			return
		}

		if chartSettings.DataportenClientSettings.Name != "" {
			regResp, err := dataporten.CreateClient(chartSettings.DataportenClientSettings, os.Getenv("TOKEN"), apiReqLogger)

			if regResp.StatusCode == http.StatusBadRequest {
				http.Error(w, regResp.Status, http.StatusBadRequest)
				return
			}

			regRes, err := dataporten.ParseRegistrationResult(regResp.Body, apiReqLogger)
			if err != nil {
				writeErrorJson(w, r, err, http.StatusInternalServerError)
				return
			}
			apiReqLogger.Debug(regRes)
		}

		res, err := install.InstallChart(chartPath, chartSettings, settings, apiReqLogger)

		if err == nil {
			render.JSON(w, r, res)
		} else {
			writeErrorJson(w, r, err, http.StatusInternalServerError)
		}
	}
}
