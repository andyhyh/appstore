package main

import (
	"context"
	"github.com/pressly/chi"
	"github.com/pressly/chi/middleware"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"net/http"
	"time"
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
	r.Get("/", listAllPackages(settings))
	r.Get("/:searchQuery", searchForPackages(settings))
	r.Post("/install/:packageName", installPackage(settings))
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

func createRoutes(settings *helm_env.EnvSettings) http.Handler {

	baseRouter := chi.NewRouter()

	baseRouter.Use(middleware.RequestID)
	baseRouter.Use(middleware.RealIP)
	baseRouter.Use(middleware.Logger)
	baseRouter.Use(middleware.Recoverer)
	baseRouter.Use(middleware.CloseNotify)
	baseRouter.Use(middleware.Timeout(60 * time.Second))
	baseRouter.Mount("/api", createAPIRouter(settings))

	return baseRouter
}
