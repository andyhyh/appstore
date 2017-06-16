package helmutil

import (
	"k8s.io/helm/pkg/helm"
	helm_env "k8s.io/helm/pkg/helm/environment"
	helm_path "k8s.io/helm/pkg/helm/helmpath"
)

func InitHelmSettings(debug bool, tillerHost string) *helm_env.EnvSettings {
	settings := new(helm_env.EnvSettings)
	settings.Home = helm_path.Home(helm_env.DefaultHelmHome())
	settings.TillerHost = tillerHost
	settings.Debug = debug

	return settings
}

func InitHelmClient(settings *helm_env.EnvSettings) helm.Interface {
	options := []helm.Option{helm.Host(settings.TillerHost)}
	// TODO: Add TLS related options.
	return helm.NewClient(options...)
}
