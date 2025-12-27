package server

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/wailsapp/wails/v3/pkg/application"

	"goprint-bridge/config"
	"goprint-bridge/logger"
	"goprint-bridge/printer"
)

// PrintRequest represents incoming print data
type PrintRequest struct {
	Type    string `json:"type"`    // text, pdf, image, etc.
	Content string `json:"content"` // Base64 encoded content or raw text
}

// PrintResponse represents the API response
type PrintResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Server holds the Fiber server instance
type Server struct {
	app      *fiber.App
	wailsApp *application.App
	mu       sync.Mutex
	running  bool
	port     int
}

var serverInstance *Server

// NewServer creates a new server instance
func NewServer(wailsApp *application.App) *Server {
	if serverInstance != nil {
		return serverInstance
	}

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
	})

	// CORS middleware - allow all origins for kiosk compatibility
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	serverInstance = &Server{
		app:      app,
		wailsApp: wailsApp,
		running:  false,
	}

	// Setup routes
	serverInstance.setupRoutes()

	return serverInstance
}

// setupRoutes configures all API endpoints
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Print endpoint
	s.app.Post("/print", func(c *fiber.Ctx) error {
		var req PrintRequest

		// Parse JSON body
		if err := c.BodyParser(&req); err != nil {
			logger.PrintError("Failed to parse print request", err)
			return c.Status(400).JSON(PrintResponse{
				Success: false,
				Message: "Invalid JSON payload",
			})
		}

		// Validate request
		if req.Type == "" || req.Content == "" {
			return c.Status(400).JSON(PrintResponse{
				Success: false,
				Message: "Missing required fields: type and content",
			})
		}

		// Get selected printer from config
		cfg := config.GetConfig()
		printerName := cfg.SelectedPrinter

		// Log the request
		logger.PrintRequest(req.Type, len(req.Content), c.IP())

		// Emit event to frontend (before printing)
		if s.wailsApp != nil {
			s.wailsApp.Event.Emit("print-received", map[string]interface{}{
				"type":    req.Type,
				"content": req.Content,
				"time":    time.Now().Format(time.RFC3339),
				"printer": printerName,
			})
		}

		// Process print job based on type
		var printErr error
		switch req.Type {
		case "pdf":
			// PDF: decode base64 and print silently
			printErr = printer.PrintPDF(printerName, req.Content)
		case "text", "raw":
			// Raw text: send directly to printer
			printErr = printer.PrintRaw(printerName, req.Content)
		default:
			// Default: treat as raw text
			printErr = printer.PrintRaw(printerName, req.Content)
		}

		if printErr != nil {
			logger.PrintError("Print job failed", printErr)

			// Emit error event
			if s.wailsApp != nil {
				s.wailsApp.Event.Emit("print-error", map[string]interface{}{
					"error": printErr.Error(),
					"time":  time.Now().Format(time.RFC3339),
				})
			}

			return c.Status(500).JSON(PrintResponse{
				Success: false,
				Message: fmt.Sprintf("Print failed: %s", printErr.Error()),
			})
		}

		// Emit success event
		if s.wailsApp != nil {
			s.wailsApp.Event.Emit("print-success", map[string]interface{}{
				"type":    req.Type,
				"printer": printerName,
				"time":    time.Now().Format(time.RFC3339),
			})
		}

		return c.JSON(PrintResponse{
			Success: true,
			Message: "Print job completed",
		})
	})
}

// Start starts the HTTP server on the specified port
func (s *Server) Start(port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server is already running")
	}

	s.port = port
	s.running = true

	// Start server in goroutine
	go func() {
		addr := fmt.Sprintf(":%d", port)
		logger.ServerStarted(port)
		if err := s.app.Listen(addr); err != nil {
			logger.PrintError("Server error", err)
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
		}
	}()

	return nil
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	logger.ServerStopped()
	s.running = false
	return s.app.Shutdown()
}

// IsRunning returns whether the server is running
func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// GetPort returns the current port
func (s *Server) GetPort() int {
	return s.port
}

// GetInstance returns the current server instance
func GetInstance() *Server {
	return serverInstance
}
