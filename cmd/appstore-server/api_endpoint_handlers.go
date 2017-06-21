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
		logger := logger.MakeAPILogger(r)
		query := chi.URLParam(r, "searchQuery")
		results, err := search.FindCharts(settings, query, "", logger)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
			apiReqLogger.Debug(fmt.Sprintf("Error decoding the POSTed JSON: '%s'", r.Body))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, "Invalid JSON!")
			return
		}

		res, err := install.InstallChart(packageName, chartSettings, settings, apiReqLogger)

		if err == nil {
			render.JSON(w, r, res)
		} else {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, err)
		}
	}
}
