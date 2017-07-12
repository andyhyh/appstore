package releaseutil

type ReleaseSettings struct {
	Repo      string                 `json:"repo"`
	Package   string                 `json:"package"`
	Version   string                 `json:"version"`
	Namespace string                 `json:"namespace"`
	Values    map[string]interface{} `json:"values"`
}

type Release struct {
	Id        string `json:"id"`
	Namespace string `json:"namespace"`
	*ReleaseSettings
}
