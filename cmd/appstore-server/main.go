package main

import (
	"flag"
	"github.com/uninett/appstore/pkg/helmutil"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"os"
)

func main() {
	debug := flag.Bool("debug", false, "Enable debug about")
	tillerHost := flag.String("host", os.Getenv(helm_env.HostEnvVar), "Enable debug about")
	flag.Parse()

	settings := helmutil.InitHelmSettings(*debug, *tillerHost)
	serveAPI(settings)
}
