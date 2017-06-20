package main

import (
	"encoding/json"
	"github.com/pressly/chi"
	"k8s.io/helm/cmd/helm/search"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPackageIndexHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(makeListAllPackagesHandler(mockSettings))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var results []*search.Result
	err = json.NewDecoder(rr.Body).Decode(&results)
	if err != nil {
		t.Errorf("decoding of result failed: %s", err.Error())
	}
}

func TestPackageSearchHandler(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/:query", makeSearchForPackagesHandler(mockSettings))

	req, err := http.NewRequest("GET", "/test", nil)

	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var results []*search.Result
	err = json.NewDecoder(rr.Body).Decode(&results)
	if err != nil {
		t.Errorf("decoding of result failed: %s", err.Error())
	}
}
