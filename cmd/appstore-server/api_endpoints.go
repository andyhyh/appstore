package main

import (
	"context"
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

func apiVersionCtx(version string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), "api.version", version))
			next.ServeHTTP(w, r)
		})
	}
}

func createPackageRouter(settings *helm_env.EnvSettings) http.Handler {
	r := chi.NewRouter()
	r.Get("/", listAllPackages(settings))
	r.Get("/:searchQuery", searchForPackages(settings))
	r.Post("/install/:packageName", installPackage(settings))
	return r
}

func CreateAPIRouter(settings *helm_env.EnvSettings) http.Handler {
	baseAPIrouter := chi.NewRouter()

	baseAPIrouter.Route("/v1", func(baseAPIrouter chi.Router) {
		baseAPIrouter.Use(apiVersionCtx("v1"))
		baseAPIrouter.Mount("/packages", createPackageRouter(settings))
	})

	return baseAPIrouter
}
