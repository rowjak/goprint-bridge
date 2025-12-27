package logger

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

// Init initializes the logger with file output
func Init() error {
	// Create storage/logs directory if not exists
	logDir := "storage/logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Open log file
	logFile := filepath.Join(logDir, "print.log")
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	// Multi-writer: console + file
	multi := io.MultiWriter(os.Stdout, file)

	// Configure zerolog
	zerolog.TimeFieldFormat = time.RFC3339
	log = zerolog.New(multi).With().Timestamp().Logger()

	return nil
}

// Info logs an info message
func Info(msg string) {
	log.Info().Msg(msg)
}

// Error logs an error message
func Error(msg string, err error) {
	log.Error().Err(err).Msg(msg)
}

// PrintRequest logs an incoming print request
func PrintRequest(contentType string, contentLength int, remoteAddr string) {
	log.Info().
		Str("type", contentType).
		Int("content_length", contentLength).
		Str("remote_addr", remoteAddr).
		Msg("Print request received")
}

// PrintSuccess logs a successful print
func PrintSuccess(printer string) {
	log.Info().
		Str("printer", printer).
		Msg("Print job sent successfully")
}

// PrintError logs a print error
func PrintError(msg string, err error) {
	log.Error().
		Err(err).
		Msg(msg)
}

// ServerStarted logs server start
func ServerStarted(port int) {
	log.Info().
		Int("port", port).
		Msg("Print server started")
}

// ServerStopped logs server stop
func ServerStopped() {
	log.Info().Msg("Print server stopped")
}
