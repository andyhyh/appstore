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

	"k8s.io/helm/pkg/helm"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/release"
)

const (
	dataportenAppstoreSettingsKey = "dataporten_appstore_settings"
	appstoreMetaDataKey           = "appstore_meta_data"
)

// Delete the release with release name releaseName.
// If the release is associated with a dataporten application, attempt to delete this as well.
func deleteReleaseHandler(releaseName string, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {
	if releaseName == "" {
		return http.StatusNotFound, fmt.Errorf("no release provided"), nil
	}
	client := helmutil.InitHelmClient(settings)

	rd, err := getReleaseDetails(releaseName, client, logger)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	status, err := client.DeleteRelease(releaseName)
	logger.Debugf("Attemping to delete: %s", releaseName)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}
	logger.Debugf("Successfully deleted: %s", releaseName)

	if dpDetailsRaw, found := rd.Values[dataportenAppstoreSettingsKey]; found {
		dpDetails := dpDetailsRaw.(map[string]interface{})
		clientId := dpDetails["id"].(string)
		logger.Debugf("Attempting to delete dataporten client: %s", clientId)

		httpResp, err := dataporten.DeleteClient(clientId, os.Getenv("TOKEN"), logger)
		if err != nil {
			return http.StatusInternalServerError, err, nil
		}
		if httpResp.StatusCode != http.StatusOK {
			return httpResp.StatusCode, fmt.Errorf(httpResp.Status), nil
		}
		logger.Debugf("Sucessfully dataporten client deleted: %s", clientId)
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

func getPackageMetaData(values map[string]interface{}) (*PackageAppstoreMetaData, error) {
	appstoreMetaDataRaw, found := values[appstoreMetaDataKey]
	if !found {
		return nil, fmt.Errorf("failed to get appstore package metadata")
	}

	appstoreMetaData, ok := appstoreMetaDataRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("package metadata is invalid")
	}

	md := new(PackageAppstoreMetaData)
	for k, v := range appstoreMetaData {
		switch k {
		case "repo":
			repo, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("invalid package metadata")
			}
			md.Repo = repo
		}
	}

	return md, nil
}

type ReleaseDetails struct {
	*release.Release
	Values           map[string]interface{}
	AppstoreMetaData *PackageAppstoreMetaData
}

func getReleaseDetails(releaseName string, client helm.Interface, logger *logrus.Entry) (*ReleaseDetails, error) {
	logger.Debugf("Attemping to fetch the details of: %s", releaseName)
	allReleaseDetails, err := client.ReleaseContent(releaseName)
	if err != nil {
		return nil, err
	}

	rel := allReleaseDetails.Release
	values := make(map[string]interface{})
	err = yaml.Unmarshal([]byte(rel.GetConfig().GetRaw()), &values)
	if err != nil {
		return nil, err
	}

	appstoreMetaData, err := getPackageMetaData(values)
	if err != nil {
		return nil, err
	}

	return &ReleaseDetails{rel, values, appstoreMetaData}, nil
}

// For the release with release name releaseName, get the same
// information about a release that was returned to the user when
// installing (i.e. the passed values etc.) the release.
func releaseDetailHandler(releaseName string, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {
	if releaseName == "" {
		return http.StatusNotFound, fmt.Errorf("no release provided"), nil
	}
	client := helmutil.InitHelmClient(settings)
	rd, err := getReleaseDetails(releaseName, client, logger)

	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	chartMetaData := rd.Chart.GetMetadata()
	if chartMetaData == nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to get chart metadata"), nil
	}

	desiredDetails := releaseutil.Release{ReleaseSettings: &releaseutil.ReleaseSettings{Repo: rd.AppstoreMetaData.Repo, Version: chartMetaData.Version, Values: rd.Values, Package: chartMetaData.Name}, Id: rd.Name, Namespace: rd.Namespace}

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

// For the release with release name releaseName, get status related
// information (i.e. whether the release is deployed, which resources it
// is using etc.)
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

// Install a release using the provided values and settings, should
// return the same values that was posted along with some extra
// information, such as which namespace it was actually deployed in etc.
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

// Upgrade the release with release name releaseName to the provided
// version (this may actually be a downgrade). The handler attempts to
// use the same repo and package name as the release was deployed with.
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
	rd, err := getReleaseDetails(releaseName, client, logger)

	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	chartMetaData := rd.Chart.GetMetadata()
	if chartMetaData == nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to get chart metadata"), nil
	}

	// TODO: Handle TLS related things:
	chartPath, err := install.LocateChartPath(chartMetaData.Name, rd.AppstoreMetaData.Repo, upgradeSettings.Version, false, "", settings, logger)
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
