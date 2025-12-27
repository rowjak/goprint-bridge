//go:build !windows
// +build !windows

package autostart

import (
	"os"
	"path/filepath"

	"github.com/emersion/go-autostart"
)

// App represents the autostart application
var app *autostart.App

// Init initializes the autostart app
func Init() {
	execPath, err := os.Executable()
	if err != nil {
		return
	}

	app = &autostart.App{
		Name:        "GoPrintBridge",
		DisplayName: "GoPrintBridge",
		Exec:        []string{execPath},
	}
}

// IsEnabled checks if autostart is enabled
func IsEnabled() bool {
	if app == nil {
		Init()
	}
	return app.IsEnabled()
}

// Enable enables autostart
func Enable() error {
	if app == nil {
		Init()
	}
	return app.Enable()
}

// Disable disables autostart
func Disable() error {
	if app == nil {
		Init()
	}
	return app.Disable()
}

// Toggle toggles autostart based on the provided state
func Toggle(enabled bool) error {
	if enabled {
		return Enable()
	}
	return Disable()
}

// GetExePath returns the executable path
func GetExePath() string {
	execPath, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Clean(execPath)
}
