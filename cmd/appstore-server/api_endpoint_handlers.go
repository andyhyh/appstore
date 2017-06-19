package main

import (
	"encoding/json"
	"fmt"
	"github.com/pressly/chi"
	"github.com/pressly/chi/render"
	"github.com/uninett/appstore/pkg/helmutil"
	"github.com/uninett/appstore/pkg/install"
	"github.com/uninett/appstore/pkg/logger"
	"github.com/uninett/appstore/pkg/search"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"net/http"
)

func makeSearchForPackagesHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := chi.URLParam(r, "searchQuery")
		results, err := search.SearchCharts(settings, query, "")

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		render.JSON(w, r, results)
	}
}

func makeListAllPackagesHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results, err := search.GetAllCharts(settings)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		newestPackages := search.GetNewestVersion(results)
		search.SortByName(newestPackages)
		render.JSON(w, r, (newestPackages))
	}
}

func makeInstallPackageHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		apiReqLogger := logger.GetApiRequestLogger(req)
		packageName := chi.URLParam(req, "packageName")
		if packageName == "" {
			http.Error(w, http.StatusText(404), 404)
			return
		}
		apiReqLogger.Debug("Installing package: " + packageName)

		chartSettings := new(helmutil.ChartSettings)
		decoder := json.NewDecoder(req.Body)
		err := decoder.Decode(&chartSettings)

		if err != nil {
			apiReqLogger.Debug(fmt.Sprintf("Error decoding the POSTed JSON: '%s'", req.Body))
			render.Status(req, http.StatusBadRequest)
			render.JSON(w, req, "Invalid JSON!")
			return
		}

		res, err := install.InstallChart(packageName, chartSettings, settings)

		if err == nil {
			render.JSON(w, req, res)
		} else {
			render.Status(req, http.StatusInternalServerError)
			render.JSON(w, req, err)
		}
	}
}
