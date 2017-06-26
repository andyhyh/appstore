package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/uninett/appstore/cmd/appstore-server/api"
	"github.com/uninett/appstore/cmd/appstore-server/dashboard"
	"github.com/uninett/appstore/pkg/helmutil"
	"github.com/uninett/appstore/pkg/logger"
	"github.com/uninett/appstore/pkg/templateutil"

	"github.com/pressly/chi"
	"github.com/pressly/chi/middleware"

	log "github.com/Sirupsen/logrus"

	helm_env "k8s.io/helm/pkg/helm/environment"
)

var startTime time.Time

const version string = "v1"

func main() {
	debug := flag.Bool("debug", false, "Enable debug about")
	port := flag.Int("port", 8080, "The port to use when hosting the server")
	tillerHost := flag.String("host", os.Getenv(helm_env.HostEnvVar), "Enable debug about")
	flag.Parse()

	settings := helmutil.InitHelmSettings(*debug, *tillerHost)

	baseRouter := chi.NewRouter()

	baseRouter.Use(middleware.RequestID)
	baseRouter.Use(middleware.RealIP)
	baseRouter.Use(logger.RequestLogger)
	baseRouter.Use(middleware.Recoverer)
	baseRouter.Use(middleware.CloseNotify)
	baseRouter.Use(middleware.Timeout(60 * time.Second))

	baseRouter.Mount("/api", api.CreateAPIRouter(settings))
	templates, err := templateutil.ProcessTemplates("ui/templates/")
	if err != nil {
		log.Fatal(err)
	}
	baseRouter.Mount("/", dashboard.CreateDashboardRouter(settings, templates))
	baseRouter.Get("/healthz", healthzHandler)
	baseRouter.FileServer("/static/", http.Dir("ui/static"))
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stderr)
	log.Debug("Starting server on port ", *port)
	log.Debug("Tiller host: ", settings.TillerHost)
	startTime = time.Now()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), baseRouter))
}
