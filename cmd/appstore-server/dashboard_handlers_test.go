package main

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
)

func initTemplates(t *testing.T) map[string]*template.Template {
	templates, err := ProcessTemplates("../../ui/templates/")
	if err != nil {
		t.Fatal(err)
	}

	if templates == nil {
		t.Fatal("could not create templates")
	}

	return templates
}

func TestTemplateProcessing(t *testing.T) {
	_ = initTemplates(t)
}

func TestDashboardPackageIndexHandler(t *testing.T) {
	templates := initTemplates(t)

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
