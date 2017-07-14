package config

import (
	"github.com/ghodss/yaml"
	"io/ioutil"
	"path/filepath"
)

type DataportenAPIConf struct {
	BasicAuthCreds    map[string]string
	GroupsEndpointUrl string
}

type AppstoreConf struct {
	DataportenConf *DataportenAPIConf
}

func LoadAppstoreConfig(confPath string) (*AppstoreConf, error) {
	filename, _ := filepath.Abs(confPath)
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	var appstoreConf *AppstoreConf
	err = yaml.Unmarshal(yamlFile, &appstoreConf)
	if err != nil {
		return nil, err
	}

	return appstoreConf, nil
}
