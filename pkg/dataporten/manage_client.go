package dataporten

import (
	"bytes"
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/m4rw3r/uuid"
	"io"
	"net/http"
	"time"
)

// 'Client' is dataporten internal name for applications.
type ClientSettings struct {
	Name            string   `json:"name"`
	ScopesRequested []string `json:"scopes_requested"`
	RedirectURI     []string `json:"redirect_uri"`
	Description     string   `json:"descr"`
	ClientSecret    string   `json:"client_secret"`
}

type RegisterClientResult struct {
	ClientSecret string `json:"client_secret"`
	ClientId     string `json:"id"`
	Owner        string `json:"owner"`
}

const dataportenURL string = "https://clientadmin.dataporten-api.no/clients/"

func initAuthorizedRequest(method string, url string, body io.Reader, token string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	return req, nil
}

func executeRequest(req *http.Request) (*http.Response, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	return client.Do(req)
}

func ParseRegistrationResult(respBody io.ReadCloser, logger *logrus.Entry) (*RegisterClientResult, error) {
	regRes := new(RegisterClientResult)
	defer respBody.Close()
	err := json.NewDecoder(respBody).Decode(&regRes)

	if err != nil {
		logger.Fatal("Dataporten returned invalid JSON " + err.Error())
		return nil, err
	}

	return regRes, nil
}

func CreateClient(cs ClientSettings, token string, logger *logrus.Entry) (*http.Response, error) {
	if cs.ClientSecret == "" {
		clientSecret, err := uuid.V4()
		if err != nil {
			logger.Debug("Could not create client secret: %s" + err.Error())
			return nil, err
		}
		cs.ClientSecret = clientSecret.String()
	}
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(cs)
	if err != nil {
		return nil, err
	}
	logger.Debug("Preparing to register new dataporten client with settings: " + b.String())

	req, err := initAuthorizedRequest("POST", dataportenURL, b, token)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	return executeRequest(req)
}

func DeleteClient(clientId string, token string, logger *logrus.Entry) (*http.Response, error) {
	deleteUrl := dataportenURL + clientId
	req, err := initAuthorizedRequest("DELETE", deleteUrl, nil, token)
	if err != nil {
		return nil, err
	}

	return executeRequest(req)
}