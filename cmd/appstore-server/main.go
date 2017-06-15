package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	restful "github.com/emicklei/go-restful"
	"github.com/uninett/appstore/pkg/helmutil"
	"github.com/uninett/appstore/pkg/search"
	"net/http"
)

const API_version string = "0.0.1"

func SearchForPackages(request *restful.Request, response *restful.Response) {
	query := request.PathParameter("search-query")
	settings := helmutil.InitHelmSettings()
	results, _ := search.SearchCharts(settings, query, "")
	response.PrettyPrint(false)
	response.WriteAsJson(results)
}

func ListAllPackages(request *restful.Request, response *restful.Response) {
	settings := helmutil.InitHelmSettings()
	results, _ := search.SearchCharts(settings, "", "")
	response.PrettyPrint(false)
	response.WriteAsJson(results)
}

func main() {
	service := new(restful.WebService)
	service.Path(fmt.Sprintf("/api/v%s", API_version)).Consumes(restful.MIME_JSON)

	service.Route(service.GET("/packages/{search-query}").To(SearchForPackages))
	service.Route(service.GET("/packages/").To(ListAllPackages))
	restful.Add(service)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
