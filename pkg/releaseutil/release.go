package releaseutil

type ReleaseSettings struct {
	Repo        string                 `json:"repo"`
	Package     string                 `json:"package"`
	Version     string                 `json:"version"`
	Namespace   string                 `json:"namespace"`
	AdminGroups []string               `json:"adminGroups"`
	Values      map[string]interface{} `json:"values"`
}

type Release struct {
	Id        string `json:"id"`
	Owner     string `json:"owner"`
	Namespace string `json:"namespace"`
	*ReleaseSettings
}
