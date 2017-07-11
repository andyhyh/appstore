package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/UNINETT/appstore/cmd/appstore-server/api"
	"github.com/UNINETT/appstore/pkg/helmutil"
	"github.com/UNINETT/appstore/pkg/logger"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"

	log "github.com/Sirupsen/logrus"

	helm_env "k8s.io/helm/pkg/helm/environment"
)

var startTime time.Time
var (
	stableRepositoryURL = "https://kubernetes-charts.storage.googleapis.com"
	// This is the IPv4 loopback, not localhost, because we have to force IPv4
	// for Dockerized Helm: https://github.com/kubernetes/helm/issues/1410
	localRepositoryURL = "http://127.0.0.1:8879/charts"
)

const version string = "v1"

func main() {
	debug := flag.Bool("debug", false, "Enable debug output")
	port := flag.Int("port", 8080, "The port to use when hosting the server")
	tillerHost := flag.String("host", os.Getenv(helm_env.HostEnvVar), "Address of tiller. Defaults to $HELM_HOST")
	flag.Parse()

	settings := helmutil.InitHelmSettings(*debug, *tillerHost)

	if err := helmutil.EnsureDirectories(settings.Home); err != nil {
		panic(err)
	}
	if err := helmutil.EnsureDefaultRepos(settings.Home, settings, false); err != nil {
		panic(err)
	}
	if err := helmutil.EnsureRepoFileFormat(settings.Home.RepositoryFile()); err != nil {
		panic(err)
	}

	baseRouter := chi.NewRouter()

	baseRouter.Use(middleware.RequestID)
	baseRouter.Use(middleware.RealIP)
	baseRouter.Use(logger.RequestLogger)
	baseRouter.Use(middleware.Recoverer)
	baseRouter.Use(middleware.CloseNotify)
	baseRouter.Use(middleware.Timeout(60 * time.Second))

	baseRouter.Mount("/api", api.CreateAPIRouter(settings))
	baseRouter.Get("/healthz", healthzHandler)

	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stderr)
	log.Debug("Starting server on port ", *port)
	log.Debug("Tiller host: ", settings.TillerHost)
	startTime = time.Now()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), baseRouter))
}
