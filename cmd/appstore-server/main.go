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
	"github.com/goware/cors"

	log "github.com/Sirupsen/logrus"

	auth "scm.uninett.no/laas/laasctl-auth"

	helm_env "k8s.io/helm/pkg/helm/environment"
)

var startTime time.Time

const version string = "v1"

func main() {
	debug := flag.Bool("debug", false, "Enable debug output")
	port := flag.Int("port", 8080, "The port to use when hosting the server")
	tillerHost := flag.String("host", os.Getenv(helm_env.HostEnvVar), "Address of tiller. Defaults to $HELM_HOST")
	flag.Parse()

	settings := helmutil.InitHelmSettings(*debug, *tillerHost)

	if settings.TillerHost == "" {
		panic(fmt.Errorf("Tiller host is missing!"))
	}

	if err := helmutil.EnsureDirectories(settings.Home); err != nil {
		panic(err)
	}
	if err := helmutil.EnsureDefaultRepos(settings.Home, settings, false); err != nil {
		panic(err)
	}
	if err := helmutil.EnsureRepoFileFormat(settings.Home.RepositoryFile()); err != nil {
		panic(err)
	}

	auth.SetConfig(
		[]string{"dataporten"},
		nil,
		map[string]string{"dataporten_creds": os.Getenv("DATAPORTEN_GK_CREDS")},
		os.Getenv("DATAPORTEN_GROUPS_ENDPOINT_URL"),
		"",
		"",
	)

	baseRouter := chi.NewRouter()

	baseRouter.Use(middleware.RequestID)
	baseRouter.Use(middleware.RealIP)
	baseRouter.Use(logger.RequestLogger)
	baseRouter.Use(middleware.Recoverer)
	baseRouter.Use(middleware.CloseNotify)
	baseRouter.Use(middleware.Timeout(60 * time.Second))

	cors := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	})
	baseRouter.Use(cors.Handler)

	baseRouter.Mount("/api", api.CreateAPIRouter(settings))
	baseRouter.Get("/healthz", healthzHandler)

	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)

	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stderr)
	log.Debug("Starting server on port ", *port)
	log.Debug("Tiller host: ", settings.TillerHost)
	startTime = time.Now()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), baseRouter))
}
