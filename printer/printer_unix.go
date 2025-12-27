//go:build !windows
// +build !windows

package printer

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"goprint-bridge/logger"
)

// PrintPDF prints a PDF file using system commands (macOS/Linux)
func PrintPDF(printerName string, base64Content string) error {
	// Decode base64 content
	pdfData, err := base64.StdEncoding.DecodeString(base64Content)
	if err != nil {
		logger.PrintError("Failed to decode base64 PDF", err)
		return fmt.Errorf("failed to decode base64: %w", err)
	}

	// Create temp file
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("goprint_%d.pdf", time.Now().UnixNano()))

	if err := os.WriteFile(tempFile, pdfData, 0644); err != nil {
		logger.PrintError("Failed to write temp PDF file", err)
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	logger.Info(fmt.Sprintf("Created temp PDF: %s", tempFile))

	// Print using lp command (CUPS)
	var cmd *exec.Cmd
	if printerName != "" {
		cmd = exec.Command("lp", "-d", printerName, tempFile)
	} else {
		cmd = exec.Command("lp", tempFile)
	}

	if err := cmd.Run(); err != nil {
		logger.PrintError("Failed to execute print command", err)
		go cleanupTempFile(tempFile, 20*time.Second)
		return fmt.Errorf("failed to print PDF: %w", err)
	}

	logger.PrintSuccess(printerName)

	// Cleanup temp file after delay
	go cleanupTempFile(tempFile, 20*time.Second)

	return nil
}

// PrintRaw prints raw text using lp command (macOS/Linux)
func PrintRaw(printerName string, content string) error {
	var cmd *exec.Cmd

	if runtime.GOOS == "darwin" {
		// macOS: Use lp with raw option
		if printerName != "" {
			cmd = exec.Command("lp", "-d", printerName, "-o", "raw")
		} else {
			cmd = exec.Command("lp", "-o", "raw")
		}
	} else {
		// Linux: Use lp
		if printerName != "" {
			cmd = exec.Command("lp", "-d", printerName)
		} else {
			cmd = exec.Command("lp")
		}
	}

	// Write content to stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	go func() {
		defer stdin.Close()
		stdin.Write([]byte(content))
	}()

	if err := cmd.Run(); err != nil {
		logger.PrintError("Failed to print raw text", err)
		return fmt.Errorf("failed to print raw text: %w", err)
	}

	logger.PrintSuccess(printerName)
	return nil
}

// PrintTestPage prints a simple test page
func PrintTestPage(printerName string) error {
	testContent := `
================================
       GOPRINT TEST PAGE
================================

Printer: %s
Time: %s

Hello World!

This is a test page from 
GoPrintBridge v1.0.0

If you can read this, your
printer is working correctly.

================================
`
	content := fmt.Sprintf(testContent, printerName, time.Now().Format("2006-01-02 15:04:05"))

	logger.Info(fmt.Sprintf("Printing test page to: %s", printerName))
	return PrintRaw(printerName, content)
}

// cleanupTempFile deletes a temporary file after a delay
func cleanupTempFile(filePath string, delay time.Duration) {
	time.Sleep(delay)
	if err := os.Remove(filePath); err != nil {
		logger.Error("Failed to cleanup temp file", err)
	} else {
		logger.Info(fmt.Sprintf("Cleaned up temp file: %s", filePath))
	}
}
