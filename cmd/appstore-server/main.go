package main

import (
	"github.com/uninett/appstore/pkg/helmutil"
)

func main() {
	settings := helmutil.InitHelmSettings()
	serveAPI(settings)
}
