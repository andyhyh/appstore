package main

import (
	"encoding/json"
	"math"
	"net/http"
	"os"
	"runtime"
	"time"
)

type HealthInfo struct {
	Version       string  `json:"version"`
	Uptime        float64 `json:"uptime"`
	Pid           int     `json:"pid"`
	NumGoroutines int     `json:"num_goroutines"`
}

func getCurrentHealth() *HealthInfo {
	hi := &HealthInfo{
		Version:       version,
		Uptime:        math.Floor(time.Since(startTime).Seconds()),
		Pid:           os.Getpid(),
		NumGoroutines: runtime.NumGoroutine(),
	}

	return hi
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	hi := getCurrentHealth()
	encoder := json.NewEncoder(w)
	encoder.Encode(hi)
}
