package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	restful "github.com/emicklei/go-restful"
	"github.com/uninett/appstore/pkg/helmutil"
	"github.com/uninett/appstore/pkg/install"
	"github.com/uninett/appstore/pkg/search"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"net/http"
)

const API_version string = "0.0.1"

func SearchForPackages(settings *helm_env.EnvSettings) func(request *restful.Request, response *restful.Response) {
	return func(request *restful.Request, response *restful.Response) {
		query := request.PathParameter("search-query")
		results, _ := search.SearchCharts(settings, query, "")
		response.WriteAsJson(results)
	}
}

func ListAllPackages(settings *helm_env.EnvSettings) func(request *restful.Request, response *restful.Response) {
	return func(request *restful.Request, response *restful.Response) {
		results, _ := search.SearchCharts(settings, "", "")
		response.WriteAsJson(results)
	}
}

func InstallPackage(settings *helm_env.EnvSettings) func(request *restful.Request, response *restful.Response) {
	return func(request *restful.Request, response *restful.Response) {
		packageName := request.PathParameter("package-name")
		log.Warn(packageName)

		chartSettings := new(helmutil.ChartSettings)
		err := request.ReadEntity(&chartSettings)

		if err != nil {
			response.WriteError(http.StatusInternalServerError, err)
		}

		settings := helmutil.InitHelmSettings()
		res, err := install.InstallChart(packageName, chartSettings, settings)

		if err == nil {
			response.WriteAsJson(res)
		} else {
			response.WriteError(http.StatusInternalServerError, err)
		}
	}
}

func main() {
	service := new(restful.WebService)
	settings := helmutil.InitHelmSettings()
	debug := false
	if debug == false {
		restful.PrettyPrintResponses = false
	}
	service.Path(fmt.Sprintf("/api/v%s", API_version)).Consumes(restful.MIME_JSON)

	service.Route(service.GET("/packages/{search-query}").To(SearchForPackages(settings)))
	service.Route(service.GET("/packages/").To(ListAllPackages(settings)))
	service.Route(service.POST("/packages/install/{package-name}").To(InstallPackage(settings)))
	restful.Add(service)
	log.Info("Starting server at port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
