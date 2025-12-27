package main

import (
	"embed"
	"log"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"goprint-bridge/config"
	"goprint-bridge/logger"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed frontend/src/assets/images/logo.png
var icon []byte

func main() {
	// Initialize logger
	if err := logger.Init(); err != nil {
		log.Printf("Failed to initialize logger: %v", err)
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Failed to load config: %v", err)
	}

	// Create the application
	app := application.New(application.Options{
		Name:        "GoPrintBridge",
		Description: "Silent Print Server",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyAccessory,
		},
	})

	// Create app service (replaces bound struct)
	appService := NewAppService(app)

	// Register service for bindings (this exposes methods to frontend)
	app.RegisterService(application.NewService(appService))

	// Create main window
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:         "GoPrintBridge",
		Width:         380,
		Height:        580,
		Hidden:        false,
		DisableResize: true,
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: false,
		},
		Mac: application.MacWindow{
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
			InvisibleTitleBarHeight: 50,
		},
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
	})

	// Handle window close - hide instead of quit
	window.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		logger.Info("Window hidden to background")
		e.Cancel()
	})

	// Create system tray (NATIVE!)
	tray := app.SystemTray.New()

	// Set tray icon
	if runtime.GOOS == "darwin" {
		// For macOS, use template icon for proper appearance in menu bar
		tray.SetTemplateIcon(icon)
	} else {
		tray.SetIcon(icon)
	}
	tray.SetTooltip("GoPrintBridge - Print Server")

	// Create tray menu
	menu := app.NewMenu()

	menu.Add("Open App").OnClick(func(ctx *application.Context) {
		// Use defer recover to prevent crash if window is destroyed
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Recovered from panic in Open App", nil)
			}
		}()
		window.Show()
		window.Focus()
	})
	menu.AddSeparator()

	// Server control menu item
	startStopItem := menu.Add("Start Server")

	// Update label based on initial state
	if appService.IsServerRunning() {
		startStopItem.SetLabel("Stop Server")
	}

	startStopItem.OnClick(func(ctx *application.Context) {
		if appService.IsServerRunning() {
			appService.StopServer()
			startStopItem.SetLabel("Start Server")
		} else {
			port := 9999
			if cfg != nil {
				port = cfg.Port
			}
			appService.StartServer(port)
			startStopItem.SetLabel("Stop Server")
		}
	})

	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(ctx *application.Context) {
		appService.Shutdown()
		app.Quit()
	})

	tray.SetMenu(menu)
	// Note: AttachWindow removed due to crash on macOS with Wails v3 alpha

	// Auto-start server if configured
	if cfg != nil && cfg.AutoStart {
		if err := appService.StartServer(cfg.Port); err != nil {
			log.Printf("Failed to auto-start server: %v", err)
		} else {
			startStopItem.SetLabel("Stop Server")
		}
	}

	logger.Info("GoPrintBridge started")

	// Run the application
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
