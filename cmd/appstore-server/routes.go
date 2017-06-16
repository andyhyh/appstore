package main

import (
	"context"
	"github.com/pressly/chi"
	"html/template"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"net/http"
)

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
	r.Get("/", makeListAllPackagesHandler(settings))
	r.Get("/:searchQuery", makeSearchForPackagesHandler(settings))
	r.Post("/install/:packageName", makeInstallPackageHandler(settings))
	return r
}

func createAPIRouter(settings *helm_env.EnvSettings) http.Handler {
	baseAPIrouter := chi.NewRouter()

	baseAPIrouter.Route("/v1", func(baseAPIrouter chi.Router) {
		baseAPIrouter.Use(apiVersionCtx("v1"))
		baseAPIrouter.Mount("/packages", createPackageRouter(settings))
	})

	return baseAPIrouter
}

func createDashboardRouter(settings *helm_env.EnvSettings, templates *template.Template) http.Handler {
	baseDashboardRouter := chi.NewRouter()

	baseDashboardRouter.Route("/", func(baseDashboardRouter chi.Router) {
		baseDashboardRouter.Get("/", makePackageIndexHandler(settings, templates))
	})

	return baseDashboardRouter
}
