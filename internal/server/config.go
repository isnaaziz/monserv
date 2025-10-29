package server

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Agents        []string
	PollInterval  time.Duration
	MemThreshold  float64
	DiskThreshold float64
	ProcThreshold float64
	LogThresholds bool
}

func LoadConfig() Config {
	agents := []string{}
	if v := os.Getenv("SERVERS"); v != "" {
		for _, s := range strings.Split(v, ",") {
			if u := strings.TrimSpace(s); u != "" {
				agents = append(agents, u)
			}
		}
	}
	poll := 5 * time.Second
	if v := os.Getenv("POLL_INTERVAL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			poll = time.Duration(n) * time.Second
		}
	}
	memTh := 90.0
	if v := os.Getenv("MEM_THRESHOLD_PERCENT"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			memTh = n
		}
	}
	diskTh := 90.0
	if v := os.Getenv("DISK_THRESHOLD_PERCENT"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			diskTh = n
		}
	}
	// Process RAM threshold (percent)
	procTh := 90.0
	if v := os.Getenv("PROC_RAM_THRESHOLD_PERCENT"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			procTh = n
		}
	}

	// Log thresholds toggle (LOG_THRESHOLDS or LOG_THRESHOLD)
	logTh := false
	if v := os.Getenv("LOG_THRESHOLDS"); v != "" {
		logTh = isTrue(v)
	}
	if v := os.Getenv("LOG_THRESHOLD"); v != "" {
		logTh = isTrue(v)
	}

	return Config{Agents: agents, PollInterval: poll, MemThreshold: memTh, DiskThreshold: diskTh, ProcThreshold: procTh, LogThresholds: logTh}
}

func isTrue(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
