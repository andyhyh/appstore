package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/ghodss/yaml"
	"github.com/go-chi/chi"

	"github.com/golang/protobuf/ptypes"

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
	appstoreMetaDataKey = "appstore_meta_data"
)

// Delete the release with release name releaseName.
// If the release is associated with a dataporten application, attempt to delete this as well.
func deleteReleaseHandler(context context.Context, releaseName string, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, interface{}, error) {
	if releaseName == "" {
		return http.StatusNotFound, nil, fmt.Errorf("no release provided")
	}
	client := helmutil.InitHelmClient(settings)

	rd, err := getReleaseDetails(releaseName, client, logger)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	status, err := client.DeleteRelease(releaseName)
	logger.Debugf("Attemping to delete: %s", releaseName)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}
	logger.Debugf("Successfully deleted: %s", releaseName)

	httpStatus, _, err := deleteClientHandler(context, rd.Values, logger)
	if err != nil {
		return httpStatus, nil, err
	}

	return http.StatusOK, status, nil
}

func makeDeleteReleaseHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		releaseName := chi.URLParam(r, "releaseName")
		status, res, err := deleteReleaseHandler(r.Context(), releaseName, settings, apiReqLogger)

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
func releaseDetailHandler(releaseName string, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, interface{}, error) {
	if releaseName == "" {
		return http.StatusNotFound, nil, fmt.Errorf("no release provided")
	}
	client := helmutil.InitHelmClient(settings)
	rd, err := getReleaseDetails(releaseName, client, logger)

	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	chartMetaData := rd.Chart.GetMetadata()
	if chartMetaData == nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to get chart metadata")
	}

	desiredDetails := releaseutil.Release{ReleaseSettings: &releaseutil.ReleaseSettings{Repo: rd.AppstoreMetaData.Repo, Version: chartMetaData.Version, Values: rd.Values, Package: chartMetaData.Name}, Id: rd.Name, Namespace: rd.Namespace}

	return http.StatusOK, desiredDetails, nil
}

func makeReleaseDetailHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		releaseName := chi.URLParam(r, "releaseName")
		status, res, err := releaseDetailHandler(releaseName, settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}

type releaseStatus struct {
	LastDeployed string                       `json:"last_deployed"`
	Namespace    string                       `json:"namespace"`
	Status       string                       `json:"status"`
	Resources    map[string]map[string]string `json:"resources"`
}

// For the release with release name releaseName, get status related
// information (i.e. whether the release is deployed, which resources it
// is using etc.)
func releaseStatusHandler(releaseName string, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, interface{}, error) {
	if releaseName == "" {
		return http.StatusNotFound, nil, fmt.Errorf("no release provided")
	}
	client := helmutil.InitHelmClient(settings)
	logger.Debugf("Attemping to fetch the status of: %s", releaseName)
	rs, err := client.ReleaseStatus(releaseName)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	info := rs.Info
	resources := releaseutil.ParseResources(info.Status.Resources)
	return http.StatusOK, releaseStatus{ptypes.TimestampString(info.GetLastDeployed()), rs.Namespace, info.Status.Code.String(), resources}, err
}

func makeReleaseStatusHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		releaseName := chi.URLParam(r, "releaseName")
		status, res, err := releaseStatusHandler(releaseName, settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}

func ReleaseOverviewHandler(settings *helm_env.EnvSettings, logger *logrus.Entry) (int, []*release.Release, error) {
	res, err := status.GetAllReleases(settings, logger)

	if err != nil {
		return http.StatusInternalServerError, nil, err
	}
	return http.StatusOK, res, nil
}

func makeReleaseOverviewHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		status, res, err := ReleaseOverviewHandler(settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}

// Install a release using the provided values and settings, should
// return the same values that was posted along with some extra
// information, such as which namespace it was actually deployed in etc.
func installReleaseHandler(context context.Context, releaseSettingsRaw io.ReadCloser, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, interface{}, error) {

	releaseSettings := &releaseutil.ReleaseSettings{Repo: "stable"}
	decoder := json.NewDecoder(releaseSettingsRaw)
	err := decoder.Decode(&releaseSettings)

	if err != nil {
		logger.Debugf("Error decoding the POSTed JSON: '%s, %s'", releaseSettingsRaw, err.Error())
		return http.StatusBadRequest, nil, fmt.Errorf("invalid json")
	}

	status, chartRequested, err := PackageDetailHandler(releaseSettings.Package, releaseSettings.Repo, releaseSettings.Version, settings, logger)
	if status != http.StatusOK {
		return status, nil, err
	}

	status, dataportenRes, err := createClientHandler(context, releaseSettings, settings, logger)
	if err != nil {
		return status, nil, err
	}
	releaseSettings.Values[dataportenAppstoreSettingsKey] = dataportenRes

	releaseSettings.Values[appstoreMetaDataKey] = PackageAppstoreMetaData{Repo: releaseSettings.Repo}
	res, err := install.InstallChart(chartRequested, releaseSettings.Namespace, releaseSettings.Values, settings, logger)

	// TODO: give a better error
	if err != nil {
		_, _, _ = deleteClientHandler(context, releaseSettings.Values, logger)
		return http.StatusOK, nil, nil
	}

	releaseSettings.Version = res.Chart.Metadata.Version
	release := releaseutil.Release{Id: res.Name, Namespace: res.Namespace, ReleaseSettings: releaseSettings}
	return http.StatusOK, release, nil
}

func makeInstallReleaseHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)

		status, res, err := installReleaseHandler(r.Context(), r.Body, settings, apiReqLogger)

		returnJSON(w, r, res, err, status)
	}
}

type UpgradeReleaseSettings struct {
	Version string `json:"version"`
}

// Upgrade the release with release name releaseName to the provided
// version (this may actually be a downgrade). The handler attempts to
// use the same repo and package name as the release was deployed with.
func upgradeReleaseHandler(releaseName string, upgradeSettingsRaw io.ReadCloser, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, interface{}, error) {
	var upgradeSettings UpgradeReleaseSettings
	decoder := json.NewDecoder(upgradeSettingsRaw)
	err := decoder.Decode(&upgradeSettings)

	if err != nil {
		logger.Debugf("Error decoding the POSTed JSON: '%s, %s'", upgradeSettingsRaw, err.Error())
		return http.StatusBadRequest, nil, fmt.Errorf("invalid json")
	}

	if releaseName == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("release not specified")
	}

	client := helmutil.InitHelmClient(settings)

	// We need some more information about the package (such as the repo
	// and package) before we can attempt to upgrade it
	rd, err := getReleaseDetails(releaseName, client, logger)

	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	chartMetaData := rd.Chart.GetMetadata()
	if chartMetaData == nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to get chart metadata")
	}

	// TODO: Handle TLS related things:
	chartPath, err := install.LocateChartPath(chartMetaData.Name, rd.AppstoreMetaData.Repo, upgradeSettings.Version, false, "", settings, logger)
	if err != nil {
		return http.StatusNotFound, nil, err
	}

	res, err := client.UpdateRelease(releaseName, chartPath)

	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return http.StatusOK, res, nil
}

func makeUpgradeReleaseHandler(settings *helm_env.EnvSettings) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiReqLogger := logger.MakeAPILogger(r)
		releaseName := chi.URLParam(r, "releaseName")
		status, res, err := upgradeReleaseHandler(releaseName, r.Body, settings, apiReqLogger)
		returnJSON(w, r, res, err, status)
	}
}
