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

const (
	levelNotice    = slog.Level(2)  // Between Info and Warn
	levelCritical  = slog.Level(10) // Between Error and Alert
	levelAlert     = slog.Level(12)
	levelEmergency = slog.Level(16) // Highest severity
)

// LogLevelMap maps string log level names to slog.Level values.
// Exported for use in validation.
var LogLevelMap = map[string]slog.Level{
	"debug":     slog.LevelDebug,
	"info":      slog.LevelInfo,
	"notice":    levelNotice,
	"warning":   slog.LevelWarn,
	"error":     slog.LevelError,
	"critical":  levelCritical,
	"alert":     levelAlert,
	"emergency": levelEmergency,
}

// levelNameMap maps slog.Level values to their uppercase string representations
var levelNameMap = map[slog.Level]string{
	slog.LevelDebug: "DEBUG",
	slog.LevelInfo:  "INFO",
	levelNotice:     "NOTICE",
	slog.LevelWarn:  "WARNING",
	slog.LevelError: "ERROR",
	levelCritical:   "CRITICAL",
	levelAlert:      "ALERT",
	levelEmergency:  "EMERGENCY",
}

// ValidLogLevels lists valid log level names (derived from LogLevelMap)
var ValidLogLevels = func() []string {
	levels := make([]string, 0, len(LogLevelMap))
	for level := range LogLevelMap {
		levels = append(levels, level)
	}
	return levels
}()

// ValidLogFormats lists valid log output formats
var ValidLogFormats = []string{"text", "json"}

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

// parseLevel converts a string to a slog.Level using the LogLevelMap.
// Supports MCP log levels: debug, info, notice, warning, error, critical, alert, emergency.
// Returns slog.LevelInfo as default if level is not recognized.
func parseLevel(level string) slog.Level {
	if lvl, ok := LogLevelMap[strings.ToLower(level)]; ok {
		return lvl
	}
	return slog.LevelInfo // default
}

func replaceAttr(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		level := a.Value.Any().(slog.Level)
		if levelName, ok := levelNameMap[level]; ok {
			a.Value = slog.StringValue(levelName)
		}
	}
	return a
}
