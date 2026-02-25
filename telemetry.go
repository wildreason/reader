package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TrackUsage logs usage to shared Punchy telemetry
// Format: timestamp|tool|subcommand
func TrackUsage(subcommand string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	telemetryDir := filepath.Join(home, ".punchy")
	telemetryFile := filepath.Join(telemetryDir, "telemetry.log")

	os.MkdirAll(telemetryDir, 0755)

	f, err := os.OpenFile(telemetryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "%d|aster|%s\n", time.Now().Unix(), subcommand)
}
