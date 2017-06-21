package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTemplateProcessing(t *testing.T) {
	templates, err := ProcessTemplates("../../ui/templates/")
	if templates == nil {
		t.Fatal("could not create templates!")
	}

	if err != nil {
		t.Fatal(err)
	}
}

func TestDashboardPackageIndexHandler(t *testing.T) {
	templates, err := ProcessTemplates("../../ui/templates/")
	if templates == nil {
		t.Fatal("could not create templates!")
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(makePackageIndexHandler(mockSettings, templates))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v (%s) want %v",
			status, rr.Body.String(), http.StatusOK)
	}
}
