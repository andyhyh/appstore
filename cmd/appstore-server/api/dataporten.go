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

func deleteClientHandler(context context.Context, vals map[string]interface{}, logger *logrus.Entry) (int, interface{}, error) {
	token := context.Value("token").(string)
	if token == "" {
		logger.Debug("No X-Dataporten-Token header not present")
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-Dataporten-Token")
	}

	dpDetailsRaw, found := vals[dataportenAppstoreSettingsKey]
	if !found {
		return http.StatusInternalServerError, nil, fmt.Errorf("Dataporten appstore settings not found")
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
		return http.StatusInternalServerError, nil, err
	}
	if httpResp.StatusCode != http.StatusOK {
		return httpResp.StatusCode, nil, fmt.Errorf(httpResp.Status)
	}

	logger.Debugf("Sucessfully deleted dataporten client: %s", clientId)
	return http.StatusOK, nil, nil

}

func createClientHandler(context context.Context, rs *releaseutil.ReleaseSettings, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, *dataporten.RegisterClientResult, error) {
	token := context.Value("token").(string)
	if token == "" {
		logger.Debug("No X-Dataporten-Token header not present")
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-Dataporten-Token")
	}

	dataportenSettings, err := dataporten.MaybeGetSettings(rs.Values)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}
	if dataportenSettings == nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("Dataporten settings missing")
	}

	logger.Debugf("Attempting to register dataporten application %s", dataportenSettings.Name)
	regResp, err := dataporten.CreateClient(dataportenSettings, token, logger)

	if regResp.StatusCode != http.StatusCreated {
		return regResp.StatusCode, nil, fmt.Errorf(regResp.Status)
	}

	dataportenRes, err := dataporten.ParseRegistrationResult(regResp.Body, logger)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	logger.Debugf("Successfully registered application %s", dataportenSettings.Name)
	return http.StatusOK, dataportenRes, nil
}
