package logger

import (
	"io"
	"log/slog"
	"strings"
)

// Service holds the logger and its dynamic level controller.
type Service struct {
	*slog.Logger
	level *slog.LevelVar
}

// Global logger instance for Phase 1 (stdio mode)
var defaultService *Service

// Define all log levels as constants following slog pattern
const (
	LevelDebug     = slog.LevelDebug // -4
	LevelInfo      = slog.LevelInfo  // 0
	LevelNotice    = slog.Level(2)   // Between Info and Warn
	LevelWarning   = slog.LevelWarn  // 4
	LevelError     = slog.LevelError // 8
	LevelCritical  = slog.Level(10)  // Between Error and Alert
	LevelAlert     = slog.Level(12)
	LevelEmergency = slog.Level(16) // Highest severity
)

// ValidLogLevels lists all valid log level names
var ValidLogLevels = []string{
	"debug",
	"info",
	"notice",
	"warning",
	"error",
	"critical",
	"alert",
	"emergency",
}

// ValidLogFormats lists valid log output formats
var ValidLogFormats = []string{"text", "json"}

// SetLevel dynamically changes the logging level for this Service instance.
func (s *Service) SetLevel(level string) {
	s.level.Set(parseLevel(level))
}

// Init initializes the global logger for Phase 1 (stdio mode).
// This sets up a default logger that can be accessed via slog package functions.
// Must be called once at application startup.
func Init(level, format string, writer io.Writer) {
	defaultService = New(level, format, writer)
	slog.SetDefault(defaultService.Logger)
}

// SetLevel changes the global log level.
// Called by MCP setLevel hook in Phase 1 (stdio mode).
func SetLevel(level string) {
	if defaultService != nil {
		defaultService.SetLevel(level)
	}
}

// New creates a new logger service with the specified configuration.
//
// Parameters:
//   - level: The logging level as a string (e.g., "debug", "info", "warn", "error").
//     See https://pkg.go.dev/log/slog#Level for more information about log levels.
//   - format: The output format, either "json" for JSON format or any other value for text format.
//   - writer: The io.Writer where log output will be written.
//
// Returns a configured *Service instance with the specified logging behavior.
func New(level, format string, writer io.Writer) *Service {
	levelVar := &slog.LevelVar{}
	levelVar.Set(parseLevel(level))

	opts := &slog.HandlerOptions{
		Level:       levelVar,
		ReplaceAttr: replaceAttr,
	}

	var handler slog.Handler
	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	default:
		handler = slog.NewTextHandler(writer, opts)
	}

	// Create the logger service
	service := &Service{
		Logger: slog.New(handler),
		level:  levelVar,
	}

	return service
}

// parseLevel converts a string to a slog.Level using a switch statement.
// Supports MCP log levels: debug, info, notice, warning, error, critical, alert, emergency.
// Returns LevelInfo as default if level is not recognized.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "notice":
		return LevelNotice
	case "warning":
		return LevelWarning
	case "error":
		return LevelError
	case "critical":
		return LevelCritical
	case "alert":
		return LevelAlert
	case "emergency":
		return LevelEmergency
	default:
		return LevelInfo
	}
}

// replaceAttr is a slog.HandlerOptions.ReplaceAttr function that customizes
// log level attribute formatting. It maps log levels to uppercase string
// representations using range-based switch cases (following slog custom levels pattern).
// All other attributes are passed through unchanged.
func replaceAttr(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		level := a.Value.Any().(slog.Level)
		switch {
		case level < LevelInfo:
			a.Value = slog.StringValue("DEBUG")
		case level < LevelNotice:
			a.Value = slog.StringValue("INFO")
		case level < LevelWarning:
			a.Value = slog.StringValue("NOTICE")
		case level < LevelError:
			a.Value = slog.StringValue("WARNING")
		case level < LevelCritical:
			a.Value = slog.StringValue("ERROR")
		case level < LevelAlert:
			a.Value = slog.StringValue("CRITICAL")
		case level < LevelEmergency:
			a.Value = slog.StringValue("ALERT")
		default:
			a.Value = slog.StringValue("EMERGENCY")
		}
	}
	return a
}
