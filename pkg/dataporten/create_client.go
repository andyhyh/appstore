package dataporten

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/m4rw3r/uuid"
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

var dataportenURL = "https://clientadmin.dataporten-api.no/clients/"

func CreateClient(cs ClientSettings, token string, logger *logrus.Entry) (*RegisterClientResult, error) {
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

	req, err := http.NewRequest("POST", dataportenURL, b)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf(resp.Status)
	}

	defer resp.Body.Close()

	regRes := new(RegisterClientResult)
	err = json.NewDecoder(resp.Body).Decode(&regRes)
	if err != nil {
		return nil, err
	}
	return regRes, nil
}

// func main() {
// name := "paal-test"
// scopes := []string{"profile"}
// redirectURI := []string{"http://example.org"}
// token := "2bb79898-1798-4118-924a-12c7692c8561"
// describ := "appstore"
// CreateDataportenClient(name, scopes, redirectURI, token, describ)
// }
