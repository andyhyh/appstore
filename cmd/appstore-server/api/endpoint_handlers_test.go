package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi"

	"github.com/UNINETT/appstore/cmd/appstore-server/handlerutil"
	"github.com/UNINETT/appstore/pkg/helmutil"

	"k8s.io/helm/cmd/helm/search"
)

func TestPackageIndexHandler(t *testing.T) {
	resp, body := handlerutil.TestHandler(t, makeListPackagesHandler(helmutil.MockSettings), "GET", "/", nil)
	handlerutil.CheckStatus(resp, http.StatusOK, t)
	var results []*Package
	err := json.NewDecoder(body).Decode(&results)
	if err != nil {
		t.Errorf("decoding of result failed: %s", err.Error())
	}
}

func TestPackageSearchHandler(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/", makeListPackagesHandler(helmutil.MockSettings))

	resp, body := handlerutil.TestHandler(t, r, "GET", "/?query=test", nil)
	handlerutil.CheckStatus(resp, http.StatusOK, t)
	var results []*search.Result
	err := json.NewDecoder(body).Decode(&results)
	if err != nil {
		t.Errorf("decoding of result failed: %s", err.Error())
	}
}
