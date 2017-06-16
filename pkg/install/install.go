package install

import (
	log "github.com/Sirupsen/logrus"
	"github.com/uninett/appstore/pkg/helmutil"
	helm_env "k8s.io/helm/pkg/helm/environment"

	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/ghodss/yaml"

	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/services"
)

type valueFiles []string

func (v *valueFiles) String() string {
	return fmt.Sprint(*v)
}

func (v *valueFiles) Type() string {
	return "valueFiles"
}

func (v *valueFiles) Set(value string) error {
	for _, filePath := range strings.Split(value, ",") {
		*v = append(*v, filePath)
	}
	return nil
}

// Merges source and destination map, preferring values from the source map
func mergeValues(dest map[string]interface{}, src map[string]interface{}) map[string]interface{} {
	for k, v := range src {
		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = v
			continue
		}
		nextMap, ok := v.(map[string]interface{})
		// If it isn't another map, overwrite the value
		if !ok {
			dest[k] = v
			continue
		}
		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = nextMap
			continue
		}
		// Edge case: If the key exists in the destination, but isn't a map
		destMap, isMap := dest[k].(map[string]interface{})
		// If the source map has a map for this key, prefer it
		if !isMap {
			dest[k] = v
			continue
		}
		// If we got to this point, it is a map in both, so merge them
		dest[k] = mergeValues(destMap, nextMap)
	}
	return dest
}

func vals(valueFiles []string) ([]byte, error) {
	base := map[string]interface{}{}

	// User specified a values files via -f/--values
	for _, filePath := range valueFiles {
		currentMap := map[string]interface{}{}
		bytes, err := ioutil.ReadFile(filePath)
		if err != nil {
			return []byte{}, err
		}

		if err := yaml.Unmarshal(bytes, &currentMap); err != nil {
			return []byte{}, fmt.Errorf("failed to parse %s: %s", filePath, err)
		}
		// Merge with the previous map
		base = mergeValues(base, currentMap)
	}

	// User specified a value via --set
	// for _, value := range i.values {
	// if err := strvals.ParseInto(value, base); err != nil {
	// return []byte{}, fmt.Errorf("failed parsing --set data: %s", err)
	// }
	// }

	return yaml.Marshal(base)
}

// locateChartPath looks for a chart directory in known places, and returns either the full path or an error.
//
// This does not ensure that the chart is well-formed; only that the requested filename exists.
//
// Order of resolution:
// - current working directory
// - if path is absolute or begins with '.', error out here
// - chart repos in $HELM_HOME
// - URL
//
// If 'verify' is true, this will attempt to also verify the chart.
func locateChartPath(name, version string, verify bool, keyring string, settings *helm_env.EnvSettings) (string, error) {
	name = "stable/" + strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if fi, err := os.Stat(name); err == nil {
		abs, err := filepath.Abs(name)
		if err != nil {
			return abs, err
		}
		if verify {
			if fi.IsDir() {
				return "", errors.New("cannot verify a directory")
			}
			if _, err := downloader.VerifyChart(abs, keyring); err != nil {
				return "", err
			}
		}
		return abs, nil
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, ".") {
		return name, fmt.Errorf("path %q not found", name)
	}

	crepo := filepath.Join(settings.Home.Repository(), name)
	if _, err := os.Stat(crepo); err == nil {
		return filepath.Abs(crepo)
	}

	dl := downloader.ChartDownloader{
		HelmHome: settings.Home,
		Out:      os.Stdout,
		Keyring:  keyring,
		Getters:  getter.All(*settings),
	}
	if verify {
		dl.Verify = downloader.VerifyAlways
	}

	filename, _, err := dl.DownloadTo(name, version, ".")
	if err == nil {
		lname, err := filepath.Abs(filename)
		if err != nil {
			return filename, err
		}
		log.Debug("Fetched %s to %s\n", name, filename)
		return lname, nil
	} else if settings.Debug {
		return filename, err
	}

	return filename, fmt.Errorf("file %q not found", name)
}

func generateName(nameTemplate string) (string, error) {
	t, err := template.New("name-template").Funcs(sprig.TxtFuncMap()).Parse(nameTemplate)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	err = t.Execute(&b, nil)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func defaultNamespace() string {
	kubeContext := ""
	if ns, _, err := kube.GetConfig(kubeContext).Namespace(); err == nil {
		return ns
	}
	return "default"
}

func checkDependencies(ch *chart.Chart, reqs *chartutil.Requirements) error {
	missing := []string{}

	deps := ch.GetDependencies()
	for _, r := range reqs.Dependencies {
		found := false
		for _, d := range deps {
			if d.Metadata.Name == r.Name {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, r.Name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("found in requirements.yaml, but missing in charts/ directory: %s", strings.Join(missing, ", "))
	}
	return nil
}

func InstallChart(chartName string, chartSettings *helmutil.ChartSettings, settings *helm_env.EnvSettings) (*services.GetReleaseStatusResponse, error) {
	client := helmutil.InitHelmClient(settings)
	namespace := ""

	// TODO: Handle TLS related things:
	cp, err := locateChartPath(chartName, chartSettings.Version, false, "", settings)
	if err != nil {
		panic(err)
	}
	chartPath := cp
	log.Debug("CHART PATH: %s\n", chartPath)

	if namespace == "" {
		namespace = defaultNamespace()
	}

	rawVals, err := vals(chartSettings.Values)
	if err != nil {
		panic(err)
	}

	// If template is specified, try to run the template.
	// if i.nameTemplate != "" {
	// i.name, err = generateName(i.nameTemplate)
	// if err != nil {
	// panic(err)
	// }
	// // Print the final name so the user knows what the final name of the release is.
	// fmt.Printf("FINAL NAME: %s\n", i.name)
	// }

	// // Check chart requirements to make sure all dependencies are present in /charts
	chartRequested, err := chartutil.Load(chartPath)
	if err != nil {
		panic(err)
	}

	if req, err := chartutil.LoadRequirements(chartRequested); err == nil {
		// If checkDependencies returns an error, we have unfullfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/kubernetes/helm/issues/2209
		if err := checkDependencies(chartRequested, req); err != nil {
			panic(err)
		}
	} else if err != chartutil.ErrRequirementsNotFound {
		log.Warn("cannot load requirements: %v", err)
		return nil, err
	}

	name := ""
	dryRun := false
	res, err := client.InstallReleaseFromChart(
		chartRequested,
		namespace,
		helm.ValueOverrides(rawVals),
		helm.ReleaseName(name),
		helm.InstallDryRun(dryRun),
		helm.InstallReuseName(false),
		helm.InstallDisableHooks(false),
		helm.InstallTimeout(0),
		helm.InstallWait(false))
	if err != nil {
		panic(err)
	}

	rel := res.GetRelease()
	if rel == nil {
		return nil, fmt.Errorf("No release returned!")
	}

	// Print the status like status command does
	status, err := client.ReleaseStatus(rel.Name)
	if err != nil && dryRun == false {
		panic(err)
	}

	return status, nil
}
