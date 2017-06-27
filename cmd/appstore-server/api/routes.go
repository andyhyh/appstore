package api

import (
	"context"
	"github.com/pressly/chi"
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

func createPackagesRouter(settings *helm_env.EnvSettings) http.Handler {
	r := chi.NewRouter()
	r.Get("/", makeListAllPackagesHandler(settings))
	r.Get("/{searchQuery}", makeSearchForPackagesHandler(settings))
	return r
}

func createPackageRouter(settings *helm_env.EnvSettings) http.Handler {
	r := chi.NewRouter()
	r.Get("/{version}/values", makePackageUserValuesHandler(settings))
	r.Post("/{version}/install", makeInstallPackageHandler(settings))
	return r
}

func CreateAPIRouter(settings *helm_env.EnvSettings) http.Handler {
	baseAPIrouter := chi.NewRouter()

	baseAPIrouter.Route("/v1", func(baseAPIrouter chi.Router) {
		baseAPIrouter.Use(apiVersionCtx("v1"))
		baseAPIrouter.Mount("/packages", createPackagesRouter(settings))
		baseAPIrouter.Mount("/package/{packageName}", createPackageRouter(settings))
	})

	return baseAPIrouter
}
