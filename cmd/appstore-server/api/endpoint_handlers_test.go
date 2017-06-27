package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/pressly/chi"

	"github.com/uninett/appstore/cmd/appstore-server/handlerutil"
	"github.com/uninett/appstore/pkg/helmutil"

	"k8s.io/helm/cmd/helm/search"
)

func TestPackageIndexHandler(t *testing.T) {
	resp, body := handlerutil.TestHandler(t, makeListAllPackagesHandler(helmutil.MockSettings), "GET", "/", nil)
	handlerutil.CheckStatus(resp, http.StatusOK, t)
	var results [][]*search.Result
	err := json.NewDecoder(body).Decode(&results)
	if err != nil {
		t.Errorf("decoding of result failed: %s", err.Error())
	}
}

func TestPackageSearchHandler(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/{query}", makeSearchForPackagesHandler(helmutil.MockSettings))

	resp, body := handlerutil.TestHandler(t, r, "GET", "/test", nil)
	handlerutil.CheckStatus(resp, http.StatusOK, t)
	var results []*search.Result
	err := json.NewDecoder(body).Decode(&results)
	if err != nil {
		t.Errorf("decoding of result failed: %s", err.Error())
	}
}
