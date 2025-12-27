//go:build windows
// +build windows

package main

import (
	"os/exec"
	"syscall"
)

// hideConsoleWindow configures the command to run without showing a console window
func hideConsoleWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
