package dataporten

import (
	"github.com/SermoDigital/jose/jws"
	"github.com/SermoDigital/jose/jwt"
	"github.com/Sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

var tokenIssuer string = os.Getenv("TOKEN_ISSUER")

func ParseRawJWT(respBody io.ReadCloser, logger *logrus.Entry) (*jwt.JWT, error) {
	token, err := ioutil.ReadAll(respBody)

	if err != nil {
		return nil, err
	}

	jwtToken, err := jws.ParseJWT(token)
	if err != nil {
		return nil, err
	}

	return &jwtToken, nil
}

func GetRawJWT(token string, logger *logrus.Entry) (*http.Response, error) {
	logger.Debugf("Attemping to get JWT token from %s", tokenIssuer)
	req, err := initAuthorizedRequest("GET", tokenIssuer, nil, token)
	if err != nil {
		return nil, err
	}

	return executeRequest(req)
}
