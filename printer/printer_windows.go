//go:build windows
// +build windows

package printer

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"goprint-bridge/logger"

	winPrinter "github.com/alexbrainman/printer"
)

// PrintPDF prints a PDF file silently using PowerShell
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

	// Print using PowerShell (silent)
	// Command: Start-Process -FilePath 'path' -Verb Print -WindowStyle Hidden
	psCmd := fmt.Sprintf(
		`Start-Process -FilePath '%s' -Verb Print -WindowStyle Hidden`,
		tempFile,
	)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	if err := cmd.Run(); err != nil {
		logger.PrintError("Failed to execute print command", err)
		// Still try to cleanup
		go cleanupTempFile(tempFile, 20*time.Second)
		return fmt.Errorf("failed to print PDF: %w", err)
	}

	logger.PrintSuccess(printerName)

	// Cleanup temp file after delay (in goroutine)
	go cleanupTempFile(tempFile, 20*time.Second)

	return nil
}

// PrintRaw prints raw text directly to the printer using Windows Spooler API
func PrintRaw(printerName string, content string) error {
	// Open printer
	p, err := winPrinter.Open(printerName)
	if err != nil {
		logger.PrintError("Failed to open printer", err)
		return fmt.Errorf("failed to open printer '%s': %w", printerName, err)
	}
	defer p.Close()

	// Start document
	if err := p.StartDocument("GoPrintBridge Document", "RAW"); err != nil {
		logger.PrintError("Failed to start document", err)
		return fmt.Errorf("failed to start document: %w", err)
	}

	// Start page
	if err := p.StartPage(); err != nil {
		logger.PrintError("Failed to start page", err)
		return fmt.Errorf("failed to start page: %w", err)
	}

	// Write content
	if _, err := p.Write([]byte(content)); err != nil {
		logger.PrintError("Failed to write to printer", err)
		return fmt.Errorf("failed to write to printer: %w", err)
	}

	// End page
	if err := p.EndPage(); err != nil {
		logger.PrintError("Failed to end page", err)
		return fmt.Errorf("failed to end page: %w", err)
	}

	// End document
	if err := p.EndDocument(); err != nil {
		logger.PrintError("Failed to end document", err)
		return fmt.Errorf("failed to end document: %w", err)
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
