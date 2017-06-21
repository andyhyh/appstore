package main

import (
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/pressly/chi"
	"github.com/pressly/chi/middleware"
	"github.com/uninett/appstore/pkg/helmutil"
	"github.com/uninett/appstore/pkg/logger"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"net/http"
	"os"
	"time"
)

var startTime time.Time
var version = "none"

func main() {
	debug := flag.Bool("debug", false, "Enable debug about")
	port := flag.Int("port", 8080, "The port to use when hosting the server")
	tillerHost := flag.String("host", os.Getenv(helm_env.HostEnvVar), "Enable debug about")
	flag.Parse()

	settings := helmutil.InitHelmSettings(*debug, *tillerHost)

	baseRouter := chi.NewRouter()
	logger.SetDebug(settings.Debug)

	baseRouter.Use(middleware.RequestID)
	baseRouter.Use(middleware.RealIP)
	baseRouter.Use(logger.Logger)
	baseRouter.Use(middleware.Recoverer)
	baseRouter.Use(middleware.CloseNotify)
	baseRouter.Use(middleware.Timeout(60 * time.Second))

	baseRouter.Mount("/api", createAPIRouter(settings))
	templates, err := ProcessTemplates("ui/templates/")
	if err != nil {
		log.Fatal(err)
	}
	baseRouter.Mount("/", createDashboardRouter(settings, templates))
	baseRouter.Get("/healthz", healthzHandler)

	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stderr)
	log.Debug("Starting server on port ", *port)
	log.Debug("Tiller host: ", settings.TillerHost)
	startTime = time.Now()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), baseRouter))
}
