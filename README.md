[![Build Status](https://travis-ci.org/UNINETT/appstore.svg?branch=master)](https://travis-ci.org/UNINETT/appstore)
### Installation guide
1. Run `make deps` to install dependencies.
2. Build the binary using `make build`.

### Running tests
1. Run `make test`

### Config
Environment variables are used for most of the config, as this makes it
easier to specify the config when running inside k8s.

The following environment variables are used:
- `HELM_HOST` is used to specify the url to the Tiller server
- `TOKEN_ISSUER` the url to the service that issues JWT tokens
- `DATAPORTEN_GK_CREDS` The basic auth credentials used by the Dataporten
  API gatekeeper
- `DATAPORTEN_GROUPS_ENDPOINT_URL` the url to the dataporten groups API
