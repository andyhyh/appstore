package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"

	"github.com/Sirupsen/logrus"
	"github.com/go-chi/chi"

	"github.com/UNINETT/appstore/pkg/dataporten"
	"github.com/UNINETT/appstore/pkg/helmutil"
	"github.com/UNINETT/appstore/pkg/install"
	"github.com/UNINETT/appstore/pkg/logger"
	"github.com/UNINETT/appstore/pkg/parseutil"
	"github.com/UNINETT/appstore/pkg/releaseutil"
	"github.com/UNINETT/appstore/pkg/status"

	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/release"
)

const dataportenAppstoreSettingsKey = "DataportenAppstoreSettings"

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
	// By default Tiller returns lone subkeys as "foo.bar: value", but
	// when parsing the JSON POSTed by the user, we expect: {"foo":
	// {"bar": "value"}} (i.e. nested objects), so we need to normalize
	// the string returned by Tiller to a format which is parsed to the
	// JSON format we expect.
	re := regexp.MustCompile(`\s*\w*(\.\w+)+: `)
	re2 := regexp.MustCompile(`(\.)`)

	normalizedConf := re.ReplaceAllStringFunc(rel.GetConfig().GetRaw(), func(m string) string {
		return re2.ReplaceAllString(m, ":\n  ")
	})
	valuesMap, err := chartutil.ReadValues([]byte(normalizedConf))

	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	desiredDetails := releaseutil.Release{ReleaseSettings: &releaseutil.ReleaseSettings{Repo: "", Version: rel.Chart.GetMetadata().Version, Namespace: rel.Namespace, Values: valuesMap}, Id: rel.Name}

	if adminGroups, found := desiredDetails.Values[dataportenAppstoreSettingsKey]; found {
		groups, err := parseutil.ParseStringList(adminGroups.([]interface{}))
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("admin groups are of wrong type"), nil
		}
		desiredDetails.AdminGroups = groups
		delete(desiredDetails.Values, dataportenAppstoreSettingsKey)
	}

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

func releaseStatusHandler(releaseName string, settings *helm_env.EnvSettings, logger *logrus.Entry) (int, error, interface{}) {
	if releaseName == "" {
		return http.StatusNotFound, fmt.Errorf("no release provided"), nil
	}
	client := helmutil.InitHelmClient(settings)
	logger.Debugf("Attemping to fetch the status of: %s", releaseName)
	status, err := client.ReleaseStatus(releaseName)
	if err != nil {
		return http.StatusInternalServerError, err, nil
	}

	return http.StatusOK, err, status
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
	var releaseSettings *releaseutil.ReleaseSettings
	decoder := json.NewDecoder(releaseSettingsRaw)
	err := decoder.Decode(&releaseSettings)

	if err != nil {
		logger.Debugf("Error decoding the POSTed JSON: '%s, %s'", releaseSettingsRaw, err.Error())
		return http.StatusBadRequest, fmt.Errorf("invalid json"), nil
	}

	status, err, chartRequested := PackageDetailHandler(releaseSettings.Package, releaseSettings.Version, settings, logger)
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

		logger.Debugf("Successfully registered application %s", dataportenSettings.Name)
	}

	if adminGroups := releaseSettings.AdminGroups; len(adminGroups) > 0 {
		releaseSettings.Values[dataportenAppstoreSettingsKey] = adminGroups
	}

	res, err := install.InstallChart(chartRequested, releaseSettings.Values, settings, logger)

	if err == nil {
		if dataportenRes != nil {
			releaseSettings.Values["dataporten"] = map[string]string{"client_id": dataportenRes.ClientId, "owner": dataportenRes.Owner}
		}
		return http.StatusOK, nil, releaseutil.Release{Id: res.Name, Owner: "", ReleaseSettings: releaseSettings}
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

// At the moment we only allow the user to up
type UpgradeReleaseSettings struct {
	Version string `json:"version"`
	Package string `json:"package"`
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

	// TODO: Handle TLS related things:
	chartPath, err := install.LocateChartPath(upgradeSettings.Package, upgradeSettings.Version, false, "", settings, logger)
	if err != nil {
		return http.StatusNotFound, err, nil
	}

	res, err := client.UpdateRelease(releaseName, chartPath, helm.ReuseValues(true), helm.UpdateValueOverrides(make([]byte, 0)))

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
