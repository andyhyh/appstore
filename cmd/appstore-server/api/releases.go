package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/ghodss/yaml"
	"github.com/go-chi/chi"

	"github.com/golang/protobuf/ptypes"

	"github.com/UNINETT/appstore/pkg/dataporten"
	"github.com/UNINETT/appstore/pkg/helmutil"
	"github.com/UNINETT/appstore/pkg/install"
	"github.com/UNINETT/appstore/pkg/logger"
	"github.com/UNINETT/appstore/pkg/releaseutil"
	"github.com/UNINETT/appstore/pkg/status"

	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/release"
)

const (
	dataportenAppstoreSettingsKey = "dataporten_appstore_settings"
	appstoreMetaDataKey           = "appstore_meta_data"
)

func deleteReleaseHandler(releaseName string, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {
	if releaseName == "" {
		return http.StatusNotFound, fmt.Errorf("no release provided"), nil
	}
	client := helmutil.InitHelmClient(settings)
	status, err := client.DeleteRelease(releaseName)
	logger.Debugf("Attemping to delete: %s", releaseName)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	return http.StatusOK, err, status
}

func makeDeleteReleaseHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		releaseName := chi.URLParam(r, "releaseName")
		status, err, res := deleteReleaseHandler(releaseName, settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}

func getPackageMetaData(values map[string]interface{}) (map[string]interface{}, error) {
	appstoreMetaDataRaw, found := values[appstoreMetaDataKey]
	if !found {
		return nil, fmt.Errorf("failed to get appstore package metadata")
	}
	appstoreMetaData, ok := appstoreMetaDataRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("package metadata is invalid")
	}

	return appstoreMetaData, nil
}

func releaseDetailHandler(releaseName string, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {
	if releaseName == "" {
		return http.StatusNotFound, fmt.Errorf("no release provided"), nil
	}
	client := helmutil.InitHelmClient(settings)
	logger.Debugf("Attemping to fetch the details of: %s", releaseName)
	allReleaseDetails, err := client.ReleaseContent(releaseName)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	rel := allReleaseDetails.Release
	valuesMap := make(map[string]interface{})
	err = yaml.Unmarshal([]byte(rel.GetConfig().GetRaw()), &valuesMap)

	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	chartMetaData := rel.Chart.GetMetadata()
	if chartMetaData == nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to get chart metadata"), nil
	}

	appstoreMetaData, err := getPackageMetaData(valuesMap)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	desiredDetails := releaseutil.Release{ReleaseSettings: &releaseutil.ReleaseSettings{Repo: appstoreMetaData["repo"].(string), Version: chartMetaData.Version, Values: valuesMap, Package: chartMetaData.Name}, Id: rel.Name, Namespace: rel.Namespace}

	return http.StatusOK, err, desiredDetails
}

func makeReleaseDetailHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		releaseName := chi.URLParam(r, "releaseName")
		status, err, res := releaseDetailHandler(releaseName, settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}

type releaseStatus struct {
	LastDeployed string                       `json:"last_deployed"`
	Namespace    string                       `json:"namespace"`
	Status       string                       `json:"status"`
	Resources    map[string]map[string]string `json:"resources"`
}

func parseResources(resources []string) map[string]map[string]string {
	parsedRes := make(map[string]map[string]string)
	for _, r := range resources {
		lines := strings.Split(strings.TrimSpace(r), "\n")
		if len(lines) > 2 {
			title := strings.TrimPrefix(lines[0], "==> ")
			col_names := strings.Fields(lines[1])
			for c_i, c_n := range col_names {
				col_names[c_i] = strings.ToLower(c_n)
			}

			items := make(map[string]string)
			for _, i := range lines[1:] {
				cols := strings.Fields(i)
				for c_i, c := range cols {
					items[col_names[c_i]] = c
				}
			}

			parsedRes[title] = items
		}
	}

	return parsedRes
}

func releaseStatusHandler(releaseName string, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {
	if releaseName == "" {
		return http.StatusNotFound, fmt.Errorf("no release provided"), nil
	}
	client := helmutil.InitHelmClient(settings)
	logger.Debugf("Attemping to fetch the status of: %s", releaseName)
	rs, err := client.ReleaseStatus(releaseName)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	info := rs.Info
	resources := parseResources(strings.Split(info.Status.Resources, "\n\n"))
	return http.StatusOK, err, releaseStatus{ptypes.TimestampString(info.GetLastDeployed()), rs.Namespace, info.Status.Code.String(), resources}
}

func makeReleaseStatusHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		releaseName := chi.URLParam(r, "releaseName")
		status, err, res := releaseStatusHandler(releaseName, settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}

func ReleaseOverviewHandler(settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, []*release.Release) {
	res, err := status.GetAllReleases(settings, logger)

	if err != nil {
		return http.StatusInternalServerError, err, nil
	}
	return http.StatusOK, nil, res
}

func makeReleaseOverviewHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		status, err, res := ReleaseOverviewHandler(settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}

func installReleaseHandler(releaseSettingsRaw io.ReadCloser, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {
	releaseSettings := &releaseutil.ReleaseSettings{Repo: "stable"}
	decoder := json.NewDecoder(releaseSettingsRaw)
	err := decoder.Decode(&releaseSettings)

	if err != nil {
		logger.Debugf("Error decoding the POSTed JSON: '%s, %s'", releaseSettingsRaw, err.Error())
		return http.StatusBadRequest, fmt.Errorf("invalid json"), nil
	}

	status, err, chartRequested := PackageDetailHandler(releaseSettings.Package, releaseSettings.Repo, releaseSettings.Version, settings, logger)
	if status != http.StatusOK {
		return status, err, nil
	}

	dataportenSettings, err := dataporten.MaybeGetSettings(releaseSettings.Values)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	var dataportenRes *dataporten.RegisterClientResult
	if dataportenSettings != nil && err == nil {
		logger.Debugf("Attempting to register dataporten application %s", dataportenSettings.Name)
		regResp, err := dataporten.CreateClient(dataportenSettings, os.Getenv("TOKEN"), logger)

		if regResp.StatusCode != http.StatusCreated {
			return regResp.StatusCode, fmt.Errorf(regResp.Status), nil
		}

		dataportenRes, err = dataporten.ParseRegistrationResult(regResp.Body, logger)
		if err != nil {
			return http.StatusInternalServerError, err, nil
		}

		releaseSettings.Values[dataportenAppstoreSettingsKey] = dataportenRes
		logger.Debugf("Successfully registered application %s", dataportenSettings.Name)
	}

	releaseSettings.Values[appstoreMetaDataKey] = PackageAppstoreMetaData{Repo: releaseSettings.Repo}
	res, err := install.InstallChart(chartRequested, releaseSettings.Namespace, releaseSettings.Values, settings, logger)

	if err == nil {
		releaseSettings.Version = res.Chart.Metadata.Version
		res := releaseutil.Release{Id: res.Name, Namespace: res.Namespace, ReleaseSettings: releaseSettings}
		return http.StatusOK, nil, res
	} else {
		if dataportenRes != nil {
			logger.Debugf("Attempting to delete dataporten client: %s", dataportenRes.ClientId)
			_, _ = dataporten.DeleteClient(dataportenRes.ClientId, os.Getenv("TOKEN"), logger)
		}
		return http.StatusInternalServerError, err, nil
	}
}

func makeInstallReleaseHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		status, err, res := installReleaseHandler(r.Body, settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}

type UpgradeReleaseSettings struct {
	Version string `json:"version"`
}

func upgradeReleaseHandler(releaseName string, upgradeSettingsRaw io.ReadCloser, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {
	var upgradeSettings UpgradeReleaseSettings
	decoder := json.NewDecoder(upgradeSettingsRaw)
	err := decoder.Decode(&upgradeSettings)

	if err != nil {
		logger.Debugf("Error decoding the POSTed JSON: '%s, %s'", upgradeSettingsRaw, err.Error())
		return http.StatusBadRequest, fmt.Errorf("invalid json"), nil
	}

	if releaseName == "" {
		return http.StatusBadRequest, fmt.Errorf("release not specified"), nil
	}

	client := helmutil.InitHelmClient(settings)

	// We need some more information about the package (such as the repo
	// and package) before we can attempt to upgrade it
	allReleaseDetails, err := client.ReleaseContent(releaseName)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	rel := allReleaseDetails.Release
	valuesMap := make(map[string]interface{})
	err = yaml.Unmarshal([]byte(rel.GetConfig().GetRaw()), &valuesMap)

	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	chartMetaData := rel.Chart.GetMetadata()
	if chartMetaData == nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to get chart metadata"), nil
	}

	appstoreMetaData, err := getPackageMetaData(valuesMap)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	// TODO: Handle TLS related things:
	chartPath, err := install.LocateChartPath(chartMetaData.Name, appstoreMetaData["repo"].(string), upgradeSettings.Version, false, "", settings, logger)
	if err != nil {
		return http.StatusNotFound, err, nil
	}

	res, err := client.UpdateRelease(releaseName, chartPath)

	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	return 200, nil, res
}

func makeUpgradeReleaseHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		releaseName := chi.URLParam(r, "releaseName")
		status, err, res := upgradeReleaseHandler(releaseName, r.Body, settings, apiReqLogger)
		returnJSON(w, r, res, err, status)
	}
}
