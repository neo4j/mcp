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

// SetLevel dynamically changes the logging level.
func (s *Service) SetLevel(level string) {
	s.level.Set(parseLevel(level))
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

const (
	levelNotice    = slog.Level(2)  // Between Info and Warn
	levelCritical  = slog.Level(10) // Between Error and Alert
	levelAlert     = slog.Level(12)
	levelEmergency = slog.Level(16) // Highest severity
)

// parseLevel converts a string to a slog.Level.
// Supports MCP log levels: debug, info, notice, warning, error, critical, alert, emergency.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "notice":
		return levelNotice
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "critical":
		return levelCritical
	case "alert":
		return levelAlert
	case "emergency":
		return levelEmergency
	default:
		return slog.LevelInfo
	}
}

func replaceAttr(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		level := a.Value.Any().(slog.Level)
		levelName := ""
		switch level {
		case slog.LevelDebug:
			levelName = "DEBUG"
		case slog.LevelInfo:
			levelName = "INFO"
		case levelNotice:
			levelName = "NOTICE"
		case slog.LevelWarn:
			levelName = "WARN"
		case slog.LevelError:
			levelName = "ERROR"
		case levelCritical:
			levelName = "CRITICAL"
		case levelAlert:
			levelName = "ALERT"
		case levelEmergency:
			levelName = "EMERGENCY"
		}
		if levelName != "" {
			a.Value = slog.StringValue(levelName)
		}
	}
	return a
}
