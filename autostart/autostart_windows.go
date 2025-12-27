//go:build windows
// +build windows

package autostart

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const (
	appName     = "GoPrintBridge"
	registryKey = `Software\Microsoft\Windows\CurrentVersion\Run`
)

// Init initializes the autostart app (no-op, kept for compatibility)
func Init() {}

// IsEnabled checks if autostart is enabled by checking the Windows registry
func IsEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, registryKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	_, _, err = key.GetStringValue(appName)
	return err == nil
}

// Enable enables autostart by adding to Windows registry
func Enable() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, registryKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	return key.SetStringValue(appName, execPath)
}

// Disable disables autostart by removing from Windows registry
func Disable() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, registryKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	return key.DeleteValue(appName)
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
