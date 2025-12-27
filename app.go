package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"goprint-bridge/autostart"
	"goprint-bridge/config"
	"goprint-bridge/logger"
	"goprint-bridge/printer"
	"goprint-bridge/server"
)

// Printer represents a printer with its status
type Printer struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

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
func (a *AppService) GetPrinters() []Printer {
	var printers []Printer

	switch runtime.GOOS {
	case "windows":
		printers = a.getWindowsPrinters()
	case "darwin":
		printers = a.getMacPrinters()
	case "linux":
		printers = a.getLinuxPrinters()
	default:
		printers = []Printer{}
	}

	return printers
}

// WindowsPrinter matches PowerShell JSON output
type WindowsPrinter struct {
	Name          string      `json:"Name"`
	PrinterStatus interface{} `json:"PrinterStatus"` // Can be string or int
}

// getWindowsPrinters gets printers using PowerShell
func (a *AppService) getWindowsPrinters() []Printer {
	// Use ConvertTo-Json to get structured data
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", "Get-Printer | Select-Object Name, PrinterStatus | ConvertTo-Json")
	hideConsoleWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		logger.Error("Failed to get Windows printers", err)
		return []Printer{}
	}

	var winPrinters []WindowsPrinter
	// If only one printer, PowerShell returns an object, not array. We might need to handle that,
	// but ConvertTo-Json usually handles arrays. However, for a single item it might return just object.
	// A trick is to wrap in @() in PowerShell: @(Get-Printer ...)
	cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", "@(Get-Printer | Select-Object Name, PrinterStatus) | ConvertTo-Json")
	hideConsoleWindow(cmd)
	output, err = cmd.Output()
	if err != nil {
		logger.Error("Failed to get Windows printers", err)
		return []Printer{}
	}

	if err := json.Unmarshal(output, &winPrinters); err != nil {
		logger.Error("Failed to parse Windows printers JSON", err)
		return []Printer{}
	}

	var result []Printer
	for _, p := range winPrinters {
		status := "Unknown"
		switch v := p.PrinterStatus.(type) {
		case string:
			status = v
		case float64:
			// Map standard Windows printer status codes
			// Ref: https://learn.microsoft.com/en-us/windows/win32/cimwin32prov/win32-printer
			code := int(v)
			switch code {
			case 1:
				status = "Other"
			case 2:
				status = "Error"
			case 3:
				status = "Ready" // Idle
			case 4:
				status = "Printing"
			case 5:
				status = "Warmup"
			case 6:
				status = "Stopped"
			case 7:
				status = "Offline"
			default:
				status = fmt.Sprintf("Status %d", code)
			}
		default:
			status = fmt.Sprintf("%v", v)
		}

		result = append(result, Printer{
			Name:   p.Name,
			Status: status,
		})
	}
	return result
}

// getMacPrinters gets printers using lpstat
func (a *AppService) getMacPrinters() []Printer {
	// Use -l for long output to get alerts/status
	cmd := exec.Command("lpstat", "-l", "-p")
	output, err := cmd.Output()
	if err != nil {
		logger.Error("Failed to get Mac printers", err)
		return []Printer{}
	}

	lines := strings.Split(string(output), "\n")
	var printers []Printer
	var currentPrinter *Printer

	// Regex to match the start of a printer block: "printer <name> is <status>."
	reStart := regexp.MustCompile(`^printer\s+(\S+)\s+is\s+(\w+)`)
	// Regex to match alert line: "Alerts: <alert>"
	reAlert := regexp.MustCompile(`^\s+Alerts:\s+(.*)`)

	for _, line := range lines {
		// Check for start of new printer block
		if matches := reStart.FindStringSubmatch(line); len(matches) >= 3 {
			// If we were processing a printer, save it
			if currentPrinter != nil {
				printers = append(printers, *currentPrinter)
			}
			status := matches[2]
			if status == "idle" {
				status = "Ready"
			} else {
				// Capitalize first letter
				if len(status) > 0 {
					status = strings.ToUpper(status[:1]) + status[1:]
				}
			}

			// Start new printer
			currentPrinter = &Printer{
				Name:   matches[1],
				Status: status,
			}
		} else if currentPrinter != nil {
			// Check for alerts in the current block
			if matches := reAlert.FindStringSubmatch(line); len(matches) >= 2 {
				alert := matches[1]
				if strings.Contains(alert, "offline") {
					currentPrinter.Status = "Offline"
				}
			}
		}
	}

	// Append the last printer
	if currentPrinter != nil {
		printers = append(printers, *currentPrinter)
	}

	return printers
}

// getLinuxPrinters gets printers using lpstat
func (a *AppService) getLinuxPrinters() []Printer {
	return a.getMacPrinters() // Same command works for Linux
}

// parseOutput is no longer used but kept for reference if needed, or can be removed.
// Removed to avoid unused code warning as getWindowsPrinters uses JSON now.

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
