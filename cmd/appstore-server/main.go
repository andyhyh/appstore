package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/uninett/appstore/pkg/helmutil"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"net/http"
	"os"
)

func main() {
	debug := flag.Bool("debug", false, "Enable debug about")
	tillerHost := flag.String("host", os.Getenv(helm_env.HostEnvVar), "Enable debug about")
	flag.Parse()

	settings := helmutil.InitHelmSettings(*debug, *tillerHost)
	router := createRoutes(settings)

	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stderr)
	log.Debug("Starting server on port 8080")
	log.Debug("Tiller host: ", settings.TillerHost)
	log.Fatal(http.ListenAndServe(":8080", router))
}
