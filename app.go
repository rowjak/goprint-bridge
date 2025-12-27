package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"goprint-bridge/autostart"
	"goprint-bridge/config"
	"goprint-bridge/logger"
	"goprint-bridge/printer"
	"goprint-bridge/server"
)

// AppService is the main application service for Wails v3
type AppService struct {
	app    *application.App
	server *server.Server
}

// NewAppService creates a new AppService instance
func NewAppService(app *application.App) *AppService {
	// Initialize autostart
	autostart.Init()

	service := &AppService{
		app: app,
	}

	// Create server instance
	service.server = server.NewServer(app)

	return service
}

// Shutdown is called when the app is closing
func (a *AppService) Shutdown() {
	if a.server != nil {
		a.server.Stop()
	}
	logger.Info("GoPrintBridge shutdown")
}

// GetPrinters returns a list of available printers
func (a *AppService) GetPrinters() []string {
	var printers []string

	switch runtime.GOOS {
	case "windows":
		printers = a.getWindowsPrinters()
	case "darwin":
		printers = a.getMacPrinters()
	case "linux":
		printers = a.getLinuxPrinters()
	default:
		printers = []string{}
	}

	return printers
}

// getWindowsPrinters gets printers using PowerShell
func (a *AppService) getWindowsPrinters() []string {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", "Get-Printer | Select-Object -ExpandProperty Name")
	hideConsoleWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		logger.Error("Failed to get Windows printers", err)
		return []string{}
	}

	return a.parseOutput(string(output))
}

// getMacPrinters gets printers using lpstat
func (a *AppService) getMacPrinters() []string {
	cmd := exec.Command("lpstat", "-p")
	output, err := cmd.Output()
	if err != nil {
		logger.Error("Failed to get Mac printers", err)
		return []string{}
	}

	lines := strings.Split(string(output), "\n")
	var printers []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "printer ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				printers = append(printers, parts[1])
			}
		}
	}

	return printers
}

// getLinuxPrinters gets printers using lpstat
func (a *AppService) getLinuxPrinters() []string {
	return a.getMacPrinters() // Same command works for Linux
}

// parseOutput cleans command output into a list
func (a *AppService) parseOutput(output string) []string {
	lines := strings.Split(output, "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

// GetConfig returns the current configuration
func (a *AppService) GetConfig() config.Config {
	cfg := config.GetConfig()
	if cfg == nil {
		return config.Config{
			SelectedPrinter: "",
			Port:            9999,
			AutoStart:       false,
		}
	}
	return *cfg
}

// SaveConfig saves the configuration
func (a *AppService) SaveConfig(printerName string, port int, autoStartEnabled bool) error {
	// Update autostart setting
	if err := autostart.Toggle(autoStartEnabled); err != nil {
		logger.Error("Failed to toggle autostart", err)
	}

	return config.UpdateConfig(printerName, port, autoStartEnabled)
}

// StartServer starts the print server
func (a *AppService) StartServer(port int) error {
	if a.server == nil {
		a.server = server.NewServer(a.app)
	}
	return a.server.Start(port)
}

// StopServer stops the print server
func (a *AppService) StopServer() error {
	if a.server == nil {
		return nil
	}
	return a.server.Stop()
}

// IsServerRunning returns whether the server is running
func (a *AppService) IsServerRunning() bool {
	if a.server == nil {
		return false
	}
	return a.server.IsRunning()
}

// PrintTestPage prints a test page to verify printer is working
func (a *AppService) PrintTestPage() error {
	cfg := config.GetConfig()
	if cfg == nil || cfg.SelectedPrinter == "" {
		return fmt.Errorf("no printer selected")
	}

	logger.Info("Printing test page to: " + cfg.SelectedPrinter)
	return printer.PrintTestPage(cfg.SelectedPrinter)
}

// GetAutoStartStatus returns whether autostart is enabled
func (a *AppService) GetAutoStartStatus() bool {
	return autostart.IsEnabled()
}

// SetAutoStart enables or disables autostart
func (a *AppService) SetAutoStart(enabled bool) error {
	return autostart.Toggle(enabled)
}

// MinimizeToTray hides the window (background mode)
func (a *AppService) MinimizeToTray() {
	// In Wails v3, we need to find the window and hide it
	// This will be handled by the window's OnClosing event
	logger.Info("Window minimized to background")
}

// ShowWindow shows the main window
func (a *AppService) ShowWindow() {
	// Window management is handled by the main.go in Wails v3
	logger.Info("Show window requested")
}

// QuitApp exits the application completely
func (a *AppService) QuitApp() {
	if a.server != nil {
		a.server.Stop()
	}
	logger.Info("Quitting application")
	a.app.Quit()
}
