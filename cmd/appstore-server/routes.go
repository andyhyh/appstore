package main

import (
	"github.com/pressly/chi"
	"github.com/pressly/chi/middleware"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"net/http"
	"time"
)

func createRoutes(settings *helm_env.EnvSettings) http.Handler {

	baseRouter := chi.NewRouter()

	baseRouter.Use(middleware.RequestID)
	baseRouter.Use(middleware.RealIP)
	baseRouter.Use(middleware.Logger)
	baseRouter.Use(middleware.Recoverer)
	baseRouter.Use(middleware.CloseNotify)
	baseRouter.Use(middleware.Timeout(60 * time.Second))
	baseRouter.Mount("/api", CreateAPIRouter(settings))

	return baseRouter
}
