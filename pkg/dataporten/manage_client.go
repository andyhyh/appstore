package dataporten

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/m4rw3r/uuid"

	"github.com/UNINETT/appstore/pkg/parseutil"
)

const dataportenClientURL string = "https://clientadmin.dataporten-api.no/clients/"

// 'Client' is dataporten internal name for applications.
type ClientSettings struct {
	Name            string   `json:"name"`
	ScopesRequested []string `json:"scopes_requested"`
	RedirectURI     []string `json:"redirect_uri"`
	Description     string   `json:"descr"`
	ClientSecret    string   `json:"client_secret"`
}

type RegisterClientResult struct {
	ClientId string   `json:"id"`
	Owner    string   `json:"owner"`
	Admins   []string `json:"admins"`
}

type DataportenClient struct {
	*RegisterClientResult
	*ClientSettings
}

func MaybeGetSettings(settings map[string]interface{}) (*ClientSettings, error) {
	secrets, found := settings["secrets"].(map[string]interface{})
	if !found {
		return nil, nil
	}

	dataportenSettings, found := secrets["dataporten"]
	if !found {
		return nil, nil
	}

	dataportenSettingsMap, ok := dataportenSettings.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("dataporten settings must be a object/map")
	}

	clientSettings := new(ClientSettings)
	if clientName, found := dataportenSettingsMap["name"]; !found {
		return nil, fmt.Errorf("dataporten name missing")
	} else {
		switch clientName.(type) {
		case string:
			clientSettings.Name = clientName.(string)
		default:
			return nil, fmt.Errorf("name must be a string")
		}
	}

	if scopesRequestedRaw, found := dataportenSettingsMap["scopes_requested"]; !found {
		return nil, fmt.Errorf("dataporten scopes missing")
	} else {
		switch scopesRequestedRaw.(type) {
		case []interface{}:
			scopesRequestedInterface := scopesRequestedRaw.([]interface{})
			scopes, err := parseutil.ParseStringList(scopesRequestedInterface)
			if err == nil {
				clientSettings.ScopesRequested = scopes
			} else {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("scopes must be a list of strings")
		}
	}

	if redirectURIRaw, found := dataportenSettingsMap["redirect_uri"]; !found {
		return nil, fmt.Errorf("dataporten redirect uri missing")
	} else {
		switch redirectURIRaw.(type) {
		case []interface{}:
			redirectURIInterface := redirectURIRaw.([]interface{})
			redirectURIs, err := parseutil.ParseStringList(redirectURIInterface)
			if err == nil {
				clientSettings.RedirectURI = redirectURIs
			} else {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("redirect URIs must be a list of strings")
		}
	}

	return clientSettings, nil
}

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

func CreateClient(cs *ClientSettings, token string, logger *logrus.Entry) (*http.Response, error) {
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

	req, err := initAuthorizedRequest("POST", dataportenClientURL, b, token)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	return executeRequest(req)
}

func DeleteClient(clientId string, token string, logger *logrus.Entry) (*http.Response, error) {
	deleteUrl := dataportenClientURL + clientId
	req, err := initAuthorizedRequest("DELETE", deleteUrl, nil, token)
	if err != nil {
		return nil, err
	}

	return executeRequest(req)
}
