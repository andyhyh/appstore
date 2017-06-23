package main

import (
	"encoding/json"
	"github.com/pressly/chi"
	"k8s.io/helm/cmd/helm/search"
	"net/http"
	"testing"
)

func TestPackageIndexHandler(t *testing.T) {
	resp, body := testHandler(t, makeListAllPackagesHandler(mockSettings), "GET", "/", nil)
	checkStatus(resp, http.StatusOK, t)
	var results [][]*search.Result
	err := json.NewDecoder(body).Decode(&results)
	if err != nil {
		t.Errorf("decoding of result failed: %s", err.Error())
	}
}

func TestPackageSearchHandler(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/:query", makeSearchForPackagesHandler(mockSettings))

	resp, body := testHandler(t, r, "GET", "/test", nil)
	checkStatus(resp, http.StatusOK, t)
	var results []*search.Result
	err := json.NewDecoder(body).Decode(&results)
	if err != nil {
		t.Errorf("decoding of result failed: %s", err.Error())
	}
}
