package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/Sirupsen/logrus"

	"github.com/UNINETT/appstore/pkg/config"
	"github.com/UNINETT/appstore/pkg/dataporten"
	"github.com/UNINETT/appstore/pkg/logger"
	helm_env "k8s.io/helm/pkg/helm/environment"
)

func listNamespacesHandler(settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {
	groupsResp, err := dataporten.RequestGroups(os.Getenv("TOKEN"), logger)
	if err != nil {
		return groupsResp.StatusCode, err, nil
	}
	userGroups, err := dataporten.ParseGroupResult(groupsResp.Body, logger)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	namespaceGroupMapping, err := config.LoadGroupMappings("./groups.yml")
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("gould not load group namespace to group mapping"), nil
	}

	allowedNamespaces := make([]*config.NamespaceMapping, 0)
	for _, n := range namespaceGroupMapping {
		for _, ag := range n.AllowedGroups {
			for _, ug := range userGroups {
				if ag == ug.GroupId {
					allowedNamespaces = append(allowedNamespaces, n)
				}
			}
		}
	}

	return http.StatusOK, nil, allowedNamespaces
}

func makeListNamespacesHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		status, err, res := listNamespacesHandler(settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}