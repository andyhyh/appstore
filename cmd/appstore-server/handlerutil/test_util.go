package handlerutil

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler(t *testing.T, h http.Handler, method, path string, body io.Reader) (*http.Response, *bytes.Buffer) {
	r, err := http.NewRequest(method, path, body)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	return w.Result(), w.Body
}

func CheckStatus(resp *http.Response, wantedStatus int, t *testing.T) {
	if status := wantedStatus; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}
