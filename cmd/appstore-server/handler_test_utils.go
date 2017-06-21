package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testHandler(t *testing.T, h http.Handler, method, path string, body io.Reader) (*http.Response, *bytes.Buffer) {
	r, err := http.NewRequest(method, path, body)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	return w.Result(), w.Body
}
