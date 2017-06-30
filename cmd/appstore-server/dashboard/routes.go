package dashboard

import (
	"github.com/go-chi/chi"
	"html/template"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"net/http"
)

func CreateDashboardRouter(settings *helm_env.EnvSettings, templates map[string]*template.Template) http.Handler {
	baseDashboardRouter := chi.NewRouter()

	// TODO: Reuse the createPackageRouter instead, and instead specify a custom writer, or something.
	baseDashboardRouter.Route("/", func(baseDashboardRouter chi.Router) {
		baseDashboardRouter.Get("/", makePackageIndexHandler(settings, templates))
		baseDashboardRouter.Get("/package/{packageName}/{version}", makePackageDetailHandler(settings, templates))
		baseDashboardRouter.Get("/releases", makeReleaseOverviewHandler(settings, templates))
	})

	return baseDashboardRouter
}
