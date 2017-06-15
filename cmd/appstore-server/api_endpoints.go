package main

import (
	"encoding/json"
	"github.com/pressly/chi"
	"github.com/pressly/chi/render"
	"github.com/uninett/appstore/pkg/helmutil"
	"github.com/uninett/appstore/pkg/install"
	"github.com/uninett/appstore/pkg/search"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"net/http"
)

func searchForPackages(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := chi.URLParam(r, "searchQuery")
		results, _ := search.SearchCharts(settings, query, "")
		render.JSON(w, r, results)
	}
}

func listAllPackages(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results, _ := search.SearchCharts(settings, "", "")
		render.JSON(w, r, results)
	}
}

func installPackage(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		packageName := chi.URLParam(req, "packageName")

		chartSettings := new(helmutil.ChartSettings)
		decoder := json.NewDecoder(req.Body)
		err := decoder.Decode(&chartSettings)
		if err != nil {
			panic(err)
		}

		if err != nil {
			render.Status(req, http.StatusInternalServerError)
			render.JSON(w, req, err)
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
