package dashboard

import (
	"html/template"
	"net/http"
	"testing"

	"github.com/uninett/appstore/pkg/helmutil"
	"github.com/uninett/appstore/pkg/templateutil"

	"github.com/uninett/appstore/cmd/appstore-server/handlerutil"
)

func initTemplates(t *testing.T) map[string]*template.Template {
	templates, err := templateutil.ProcessTemplates("../../ui/templates/")
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

	resp, _ := handlerutil.TestHandler(t, makePackageIndexHandler(helmutil.MockSettings, templates), "GET", "/", nil)
	handlerutil.CheckStatus(resp, http.StatusOK, t)
}
