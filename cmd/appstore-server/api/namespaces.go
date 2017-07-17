package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"

	"github.com/UNINETT/appstore/pkg/config"
	"github.com/UNINETT/appstore/pkg/dataporten"
	"github.com/UNINETT/appstore/pkg/logger"
	helm_env "k8s.io/helm/pkg/helm/environment"
)

const (
	namespaceMappingFile = "subjects.yml"
)

// Return a list of the namespaces the enduser is allowed to deploy to.
// The namespaceMappingFile contains a hardcoded mapping between
// namespaces and subjects (which in this case may be dataporten
// groups), and this mapping is used to determine which namespace the
// user is allowed to use.
func listNamespacesHandler(context context.Context, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, interface{}, error) {
	token := context.Value("token").(string)
	if token == "" {
		logger.Debug("No X-Dataporten-Token header not present")
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-Dataporten-Token")
	}
	groupsResp, err := dataporten.RequestGroups(token, logger)
	if err != nil {
		return groupsResp.StatusCode, nil, err
	}
	if groupsResp.StatusCode != 200 {
		return groupsResp.StatusCode, nil, fmt.Errorf(groupsResp.Status)
	}
	userGroups, err := dataporten.ParseGroupResult(groupsResp.Body, logger)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	namespaceSubjectMapping, err := config.LoadNamespaceMappings("./" + namespaceMappingFile)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("could not load namespace to subject mapping")
	}

	allowedNamespaces := make([]*config.NamespaceMapping, 0)
	for _, n := range namespaceSubjectMapping {
		for _, ag := range n.AllowedSubjects {
			for _, ug := range userGroups {
				if ag == ug.GroupId {
					allowedNamespaces = append(allowedNamespaces, n)
				}
			}
		}
	}

	return http.StatusOK, allowedNamespaces, nil
}

func makeListNamespacesHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		status, res, err := listNamespacesHandler(r.Context(), settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}
