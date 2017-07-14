package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"

	"github.com/UNINETT/appstore/pkg/dataporten"
	"github.com/UNINETT/appstore/pkg/releaseutil"

	helm_env "k8s.io/helm/pkg/helm/environment"
)

const (
	dataportenAppstoreSettingsKey = "dataporten_appstore_settings"
)

func deleteClientHandler(vals map[string]interface{}, context context.Context, logger *logrus.Entry) (int, error, interface{}) {
	token := context.Value("token").(string)
	if token == "" {
		logger.Debug("No X-Dataporten-Token header not present")
		return http.StatusBadRequest, fmt.Errorf("missing X-Dataporten-Token"), nil
	}

	dpDetailsRaw, found := vals[dataportenAppstoreSettingsKey]
	if !found {
		return http.StatusInternalServerError, fmt.Errorf("Dataporten appstore settings not found"), nil
	}
	var clientId string
	switch dpDetailsRaw.(type) {
	case map[string]interface{}:
		dpDetails := dpDetailsRaw.(map[string]interface{})
		clientId = dpDetails["id"].(string)
	case *dataporten.RegisterClientResult:
		dpDetails := dpDetailsRaw.(*dataporten.RegisterClientResult)
		clientId = dpDetails.ClientId
	}

	logger.Debugf("Attempting to delete dataporten client: %s", clientId)
	httpResp, err := dataporten.DeleteClient(clientId, token, logger)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}
	if httpResp.StatusCode != http.StatusOK {
		return httpResp.StatusCode, fmt.Errorf(httpResp.Status), nil
	}

	logger.Debugf("Sucessfully deleted dataporten client: %s", clientId)
	return http.StatusOK, nil, nil

}

func createClientHandler(rs *releaseutil.ReleaseSettings, context context.Context, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, *dataporten.RegisterClientResult) {
	token := context.Value("token").(string)
	if token == "" {
		logger.Debug("No X-Dataporten-Token header not present")
		return http.StatusBadRequest, fmt.Errorf("missing X-Dataporten-Token"), nil
	}

	dataportenSettings, err := dataporten.MaybeGetSettings(rs.Values)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}
	if dataportenSettings == nil {
		return http.StatusInternalServerError, fmt.Errorf("Dataporten settings missing"), nil
	}

	logger.Debugf("Attempting to register dataporten application %s", dataportenSettings.Name)
	regResp, err := dataporten.CreateClient(dataportenSettings, token, logger)

	if regResp.StatusCode != http.StatusCreated {
		return regResp.StatusCode, fmt.Errorf(regResp.Status), nil
	}

	dataportenRes, err := dataporten.ParseRegistrationResult(regResp.Body, logger)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	logger.Debugf("Successfully registered application %s", dataportenSettings.Name)
	return http.StatusOK, nil, dataportenRes
}
