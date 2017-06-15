package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/pressly/chi"
	"github.com/pressly/chi/middleware"
	"github.com/uninett/appstore/pkg/helmutil"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"net/http"
	"os"
	"time"
)

func main() {
	debug := flag.Bool("debug", false, "Enable debug about")
	tillerHost := flag.String("host", os.Getenv(helm_env.HostEnvVar), "Enable debug about")
	flag.Parse()

	settings := helmutil.InitHelmSettings(*debug, *tillerHost)

	baseRouter := chi.NewRouter()

	baseRouter.Use(middleware.RequestID)
	baseRouter.Use(middleware.RealIP)
	baseRouter.Use(middleware.Logger)
	baseRouter.Use(middleware.Recoverer)
	baseRouter.Use(middleware.CloseNotify)
	baseRouter.Use(middleware.Timeout(60 * time.Second))

	baseRouter.Mount("/api", createAPIRouter(settings))

	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stderr)
	log.Debug("Starting server on port 8080")
	log.Debug("Tiller host: ", settings.TillerHost)
	log.Fatal(http.ListenAndServe(":8080", baseRouter))
}
