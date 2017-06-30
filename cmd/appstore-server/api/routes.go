package api

import (
	"context"
	"github.com/go-chi/chi"
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

func createNamespacesRouter(settings *helm_env.EnvSettings) http.Handler {
	r := chi.NewRouter()
	r.Get("/", makeListNamespacesHandler(settings))
	return r
}

func createPackagesRouter(settings *helm_env.EnvSettings) http.Handler {
	r := chi.NewRouter()
	r.Get("/", makeListPackagesHandler(settings))
	return r
}

func createReleaseRouter(settings *helm_env.EnvSettings) http.Handler {
	r := chi.NewRouter()
	r.Get("/", makeReleaseOverviewHandler(settings))
	r.Post("/", makeInstallReleaseHandler(settings))
	r.Get("/{releaseName}", makeReleaseDetailHandler(settings))
	r.Get("/{releaseName}/status", makeReleaseStatusHandler(settings))
	r.Patch("/{releaseName}", makeUpgradeReleaseHandler(settings))
	r.Delete("/{releaseName}", makeDeleteReleaseHandler(settings))
	return r
}

func CreateAPIRouter(settings *helm_env.EnvSettings) http.Handler {
	baseAPIrouter := chi.NewRouter()

	baseAPIrouter.Route("/v1", func(baseAPIrouter chi.Router) {
		baseAPIrouter.Use(apiVersionCtx("v1"))
		baseAPIrouter.Mount("/packages", createPackagesRouter(settings))
		baseAPIrouter.Mount("/releases", createReleaseRouter(settings))
		baseAPIrouter.Mount("/namespaces", createNamespacesRouter(settings))
	})

	return baseAPIrouter
}
