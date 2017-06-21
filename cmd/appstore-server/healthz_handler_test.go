package main

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

func TestHealthzHandler(t *testing.T) {
	resp, body := testHandler(t, http.HandlerFunc(healthzHandler), "GET", "/health", nil)

	if status := resp.StatusCode; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var returnedHi HealthInfo
	err := json.NewDecoder(body).Decode(&returnedHi)
	if err != nil {
		t.Errorf("handler returned invalid JSON!")
	}

	expectedPid := os.Getpid()
	if returnedHi.Pid != expectedPid {
		t.Errorf("handler returned unexpected pid: got %v want %v",
			returnedHi.Pid, expectedPid)
	}
}
