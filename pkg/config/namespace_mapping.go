package config

import (
	"github.com/ghodss/yaml"
	"io/ioutil"
	"path/filepath"
)

type NamespaceMapping struct {
	NamespaceId     string   `json:"id"`
	Description     string   `json:"description"`
	AllowedSubjects []string `json:"subjects"`
}

func LoadNamespaceMappings(yamlFilepath string) ([]*NamespaceMapping, error) {
	filename, _ := filepath.Abs(yamlFilepath)
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	var namespaceMapping []*NamespaceMapping
	err = yaml.Unmarshal(yamlFile, &namespaceMapping)
	if err != nil {
		return nil, err
	}

	return namespaceMapping, nil
}
