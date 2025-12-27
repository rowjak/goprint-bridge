//go:build !windows
// +build !windows

package main

import "os/exec"

// hideConsoleWindow is a no-op on non-Windows platforms
func hideConsoleWindow(cmd *exec.Cmd) {
	// No action needed on non-Windows
}
