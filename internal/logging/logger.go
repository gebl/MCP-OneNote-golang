// logging.go - Centralized logging configuration for the OneNote MCP server.
//
// This package provides a structured logging solution using Go's slog package
// with configurable log levels, structured output, and component-based loggers.
//
// Key Features:
// - Structured logging with key-value pairs
// - Configurable log levels (DEBUG, INFO, WARN, ERROR)
// - Component-based loggers with automatic prefixing
// - Environment-based configuration
// - Performance optimized with lazy evaluation
// - Support for both text and JSON output formats
//
// Usage:
//   logger := logging.GetLogger("auth")
//   logger.Info("Authentication started", "user_id", userID)
//   logger.Debug("Token details", "expires_in", tokenExpiry)
//   logger.Error("Authentication failed", "error", err)
//
// Configuration:
// - LOG_LEVEL: Set to DEBUG, INFO, WARN, or ERROR (default: INFO)
// - LOG_FORMAT: Set to "json" for JSON output, "text" for human-readable (default: text)
// - MCP_LOG_FILE: Optional file path for log output
// - CONTENT_LOG_LEVEL: Set verbosity for content logging - DEBUG, INFO, WARN, ERROR, or OFF (default: DEBUG during development)

package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

var (
	defaultLogger   *slog.Logger
	logLevel        slog.Level = slog.LevelDebug // Start with maximum verbosity until config is loaded
	contentLogLevel slog.Level = slog.LevelDebug // Default to DEBUG during development
)

// Initialize sets up the global logger configuration based on environment variables
// This is used for early initialization before config loading
// IMPORTANT: Defaults to DEBUG level to capture all config loading details
func Initialize() {
	InitializeFromEnv()
}

// InitializeFromEnv sets up logging from environment variables only
// Used during early startup before config object is available
// IMPORTANT: Defaults to DEBUG level to capture all config loading details
func InitializeFromEnv() {
	// Determine log level from environment
	levelStr := strings.ToUpper(os.Getenv("LOG_LEVEL"))
	switch levelStr {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "INFO":
		logLevel = slog.LevelInfo
	case "WARN", "WARNING":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	default:
		// Default to DEBUG during early initialization to capture config loading details
		logLevel = slog.LevelDebug
	}

	// Determine content logging level from environment
	contentLevelStr := strings.ToUpper(os.Getenv("CONTENT_LOG_LEVEL"))
	switch contentLevelStr {
	case "DEBUG":
		contentLogLevel = slog.LevelDebug
	case "INFO":
		contentLogLevel = slog.LevelInfo
	case "WARN", "WARNING":
		contentLogLevel = slog.LevelWarn
	case "ERROR":
		contentLogLevel = slog.LevelError
	case "OFF":
		contentLogLevel = slog.Level(1000) // Very high level to effectively disable
	default:
		contentLogLevel = slog.LevelDebug // Default to DEBUG during development
	}

	// Determine output destination
	var output io.Writer = os.Stderr
	if logFile := os.Getenv("MCP_LOG_FILE"); logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			// Fallback to stderr and log the error
			slog.Error("Failed to open log file, using stderr", "file", logFile, "error", err)
		} else {
			output = file
		}
	}

	// Determine log format
	var handler slog.Handler
	format := strings.ToLower(os.Getenv("LOG_FORMAT"))
	if format == "json" {
		handler = slog.NewJSONHandler(output, &slog.HandlerOptions{
			Level: logLevel,
		})
	} else {
		handler = slog.NewTextHandler(output, &slog.HandlerOptions{
			Level: logLevel,
		})
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

// LoggingConfig interface defines the structure for logging configuration
type LoggingConfig interface {
	GetLogLevel() string
	GetLogFormat() string
	GetLogFile() string
	GetContentLogLevel() string
}

// InitializeFromConfig reinitializes logging based on configuration object
// This is called after config loading to apply any settings from config files
// Note: This may reduce verbosity from the initial DEBUG level used during config loading
func InitializeFromConfig(cfg LoggingConfig) {
	// Log the transition from initial config loading verbosity to final config
	if defaultLogger != nil {
		defaultLogger.Debug("Transitioning from config loading verbosity to final logging configuration")
	}

	// Use config values, falling back to environment variables if config values are empty
	logLevelStr := cfg.GetLogLevel()
	if logLevelStr == "" {
		logLevelStr = os.Getenv("LOG_LEVEL")
	}

	logFormatStr := cfg.GetLogFormat()
	if logFormatStr == "" {
		logFormatStr = os.Getenv("LOG_FORMAT")
	}

	logFileStr := cfg.GetLogFile()
	if logFileStr == "" {
		logFileStr = os.Getenv("MCP_LOG_FILE")
	}

	contentLogLevelStr := cfg.GetContentLogLevel()
	if contentLogLevelStr == "" {
		contentLogLevelStr = os.Getenv("CONTENT_LOG_LEVEL")
	}

	// Parse log level
	switch strings.ToUpper(logLevelStr) {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "INFO":
		logLevel = slog.LevelInfo
	case "WARN", "WARNING":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	default:
		// After config loading, default to INFO level (less verbose than initial DEBUG)
		logLevel = slog.LevelInfo
	}

	// Parse content log level
	switch strings.ToUpper(contentLogLevelStr) {
	case "DEBUG":
		contentLogLevel = slog.LevelDebug
	case "INFO":
		contentLogLevel = slog.LevelInfo
	case "WARN", "WARNING":
		contentLogLevel = slog.LevelWarn
	case "ERROR":
		contentLogLevel = slog.LevelError
	case "OFF":
		contentLogLevel = slog.Level(1000) // Very high level to effectively disable
	default:
		contentLogLevel = slog.LevelDebug // Default to DEBUG during development
	}

	// Determine output destination
	var output io.Writer = os.Stderr
	if logFileStr != "" {
		file, err := os.OpenFile(logFileStr, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			// Fallback to stderr and log the error
			if defaultLogger != nil {
				defaultLogger.Error("Failed to open log file, using stderr", "file", logFileStr, "error", err)
			}
		} else {
			output = file
		}
	}

	// Determine log format
	var handler slog.Handler
	if strings.ToLower(logFormatStr) == "json" {
		handler = slog.NewJSONHandler(output, &slog.HandlerOptions{
			Level: logLevel,
		})
	} else {
		handler = slog.NewTextHandler(output, &slog.HandlerOptions{
			Level: logLevel,
		})
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)

	// Log the final logging configuration
	defaultLogger.Debug("Logging reconfigured from config",
		"final_log_level", logLevel.String(),
		"final_content_log_level", contentLogLevel.String(),
		"log_format", strings.ToLower(logFormatStr),
		"log_file", logFileStr,
		"config_source", "configuration_object")
}

// GetLogger returns a component-specific logger with the given component name
// The component name will be included in all log entries for easier filtering
func GetLogger(component string) *slog.Logger {
	if defaultLogger == nil {
		Initialize()
	}
	return defaultLogger.With("component", component)
}

// GetLevel returns the current log level
func GetLevel() slog.Level {
	return logLevel
}

// IsDebugEnabled returns true if debug logging is enabled
func IsDebugEnabled() bool {
	return logLevel <= slog.LevelDebug
}

// IsContentLoggingEnabled returns true if content logging is enabled at the specified level
func IsContentLoggingEnabled(level slog.Level) bool {
	return contentLogLevel <= level
}

// GetContentLogLevel returns the current content log level
func GetContentLogLevel() slog.Level {
	return contentLogLevel
}

// SetLevel sets the log level programmatically (useful for testing)
func SetLevel(level slog.Level) {
	logLevel = level
	Initialize() // Reinitialize with new level
}

// SetContentLogLevel sets the content log level programmatically
func SetContentLogLevel(level slog.Level) {
	contentLogLevel = level
}

// LogContent conditionally logs content based on the content logging configuration
// This is a convenience function to avoid checking content logging level in every location
func LogContent(logger *slog.Logger, level slog.Level, msg string, args ...any) {
	if IsContentLoggingEnabled(level) {
		logger.Log(context.Background(), level, msg, args...)
	}
}

// Component-specific logger instances for commonly used components
var (
	AuthLogger     = GetLogger("auth")
	ConfigLogger   = GetLogger("config")
	GraphLogger    = GetLogger("graph")
	NotebookLogger = GetLogger("notebook")
	PageLogger     = GetLogger("page")
	SectionLogger  = GetLogger("section")
	ToolsLogger    = GetLogger("tools")
	MainLogger     = GetLogger("main")
)
