package helmutil

import (
	"github.com/uninett/appstore/pkg/dataporten"
)

type ChartSettings struct {
	Version                  string                    `json:"version"`
	Values                   map[string]string         `json:"values"`
	DataportenClientSettings dataporten.ClientSettings `json:"dataporten_client_settings"`
}
