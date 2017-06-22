package main

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

func TestHealthzHandler(t *testing.T) {
	resp, body := testHandler(t, http.HandlerFunc(healthzHandler), "GET", "/health", nil)

	checkStatus(resp, http.StatusOK, t)

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
