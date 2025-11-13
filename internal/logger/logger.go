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

// New creates a new logging service.
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
	LevelNotice    = slog.Level(2)  // Between Info and Warn
	LevelCritical  = slog.Level(10) // Between Error and Alert
	LevelAlert     = slog.Level(12)
	LevelEmergency = slog.Level(16) // Highest severity
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
		return LevelNotice
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "critical":
		return LevelCritical
	case "alert":
		return LevelAlert
	case "emergency":
		return LevelEmergency
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
		case LevelNotice:
			levelName = "NOTICE"
		case slog.LevelWarn:
			levelName = "WARN"
		case slog.LevelError:
			levelName = "ERROR"
		case LevelCritical:
			levelName = "CRITICAL"
		case LevelAlert:
			levelName = "ALERT"
		case LevelEmergency:
			levelName = "EMERGENCY"
		}
		if levelName != "" {
			a.Value = slog.StringValue(levelName)
		}
	}
	return a
}
