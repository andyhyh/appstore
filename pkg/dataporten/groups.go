package dataporten

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"io"
	"net/http"
)

const dataportenGroupURL string = "https://groups-api.dataporten.no/groups/"

// 'Client' is dataporten internal name for applications.
type DataportenGroup struct {
	GroupId string `json:"id"`
}

func ParseGroupResult(respBody io.ReadCloser, logger *logrus.Entry) ([]*DataportenGroup, error) {
	var groups []*DataportenGroup
	defer respBody.Close()
	err := json.NewDecoder(respBody).Decode(&groups)

	if err != nil {
		logger.Debug("Dataporten returned invalid JSON " + err.Error())
		return nil, err
	}

	return groups, nil
}

func RequestGroups(token string, logger *logrus.Entry) (*http.Response, error) {
	req, err := initAuthorizedRequest("GET", dataportenGroupURL+"me/groups", nil, token)
	if err != nil {
		return nil, err
	}

	return executeRequest(req)
}
