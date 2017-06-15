package helmutil

import (
	helm_env "k8s.io/helm/pkg/helm/environment"
	helm_path "k8s.io/helm/pkg/helm/helmpath"
)

func InitHelmSettings() helm_env.EnvSettings {
	var settings helm_env.EnvSettings
	settings.Home = helm_path.Home(helm_env.DefaultHelmHome)

	return settings
}
