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

	if status := resp.StatusCode; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var results []*search.Result
	err := json.NewDecoder(body).Decode(&results)
	if err != nil {
		t.Errorf("decoding of result failed: %s", err.Error())
	}
}

func TestPackageSearchHandler(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/:query", makeSearchForPackagesHandler(mockSettings))

	resp, body := testHandler(t, r, "GET", "/test", nil)

	if status := resp.StatusCode; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var results []*search.Result
	err := json.NewDecoder(body).Decode(&results)
	if err != nil {
		t.Errorf("decoding of result failed: %s", err.Error())
	}
}
