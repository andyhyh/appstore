package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHealthzHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthzHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var returnedHi HealthInfo
	requestBody := rr.Body
	err = json.NewDecoder(requestBody).Decode(&returnedHi)
	if err != nil {
		t.Errorf("handler returned invalid JSON!")
	}

	expectedPid := os.Getpid()
	if returnedHi.Pid != expectedPid {
		t.Errorf("handler returned unexpected pid: got %v want %v",
			returnedHi.Pid, expectedPid)
	}
}
