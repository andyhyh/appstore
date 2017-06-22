package main

import (
	"html/template"
	"net/http"
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

	resp, _ := testHandler(t, makePackageIndexHandler(mockSettings, templates), "GET", "/", nil)
	checkStatus(resp, http.StatusOK, t)
}
