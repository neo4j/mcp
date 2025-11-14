package logger_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/neo4j/mcp/internal/logger"
)

func TestDynamicLogLevelChange(t *testing.T) {
	t.Run("changing log level from info to debug shows debug logs", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New("info", "text", buf)

		// At info level, debug logs should NOT appear
		log.Debug("debug message")
		log.Info("info message")

		output := buf.String()
		if strings.Contains(output, "debug message") {
			t.Error("Expected debug message to NOT appear at info level")
		}
		if !strings.Contains(output, "info message") {
			t.Error("Expected info message to appear at info level")
		}

		// Now change to debug level
		buf.Reset()
		log.SetLevel("debug")
		log.Debug("debug message after change")
		log.Info("info message after change")

		output = buf.String()
		if !strings.Contains(output, "debug message after change") {
			t.Error("Expected debug message to appear after changing to debug level")
		}
		if !strings.Contains(output, "info message after change") {
			t.Error("Expected info message to appear after changing to debug level")
		}
	})

	t.Run("changing log level from debug to error filters info/debug logs", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New("debug", "text", buf)

		// At debug level, all logs should appear
		log.Debug("debug message")
		log.Info("info message")
		log.Error("error message")

		output := buf.String()
		if !strings.Contains(output, "debug message") {
			t.Error("Expected debug message to appear at debug level")
		}
		if !strings.Contains(output, "info message") {
			t.Error("Expected info message to appear at debug level")
		}
		if !strings.Contains(output, "error message") {
			t.Error("Expected error message to appear at debug level")
		}

		// Now change to error level
		buf.Reset()
		log.SetLevel("error")
		log.Debug("debug after error level")
		log.Info("info after error level")
		log.Error("error after error level")

		output = buf.String()
		if strings.Contains(output, "debug after error level") {
			t.Error("Expected debug message to NOT appear at error level")
		}
		if strings.Contains(output, "info after error level") {
			t.Error("Expected info message to NOT appear at error level")
		}
		if !strings.Contains(output, "error after error level") {
			t.Error("Expected error message to appear at error level")
		}
	})

	t.Run("log level strings are case insensitive", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New("info", "text", buf)

		buf.Reset()
		log.SetLevel("DEBUG")
		log.Debug("debug message uppercase")
		output := buf.String()
		if !strings.Contains(output, "debug message uppercase") {
			t.Error("Expected DEBUG (uppercase) to change log level to debug")
		}

		buf.Reset()
		log.SetLevel("Error")
		log.Error("error message mixed case")
		log.Info("info should not appear")
		output = buf.String()
		if !strings.Contains(output, "error message mixed case") {
			t.Error("Expected Error (mixed case) to change log level to error")
		}
		if strings.Contains(output, "info should not appear") {
			t.Error("Expected info to NOT appear at error level")
		}
	})

	t.Run("all log levels can be set dynamically", func(t *testing.T) {
		levels := []string{"debug", "info", "notice", "warn", "warning", "error", "critical", "alert", "emergency"}

		for _, lvl := range levels {
			buf := &bytes.Buffer{}
			log := logger.New("debug", "text", buf)

			log.SetLevel(lvl)
			log.Debug("test debug")
			log.Info("test info")
			log.Error("test error")

			// Just verify SetLevel doesn't panic
			if t.Failed() {
				t.Errorf("SetLevel(%q) caused test to fail", lvl)
			}
		}
	})

	t.Run("json format with dynamic log level changes", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New("info", "json", buf)

		log.Info("info message")
		output := buf.String()

		// Validate the output is valid JSON with expected fields
		var logEntry map[string]any
		if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
			t.Errorf("Expected valid JSON output, got: %v (output: %s)", err, output)
		}
		if _, hasLevel := logEntry["level"]; !hasLevel {
			t.Error("Expected JSON output to contain 'level' field")
		}
		if msg, hasMsg := logEntry["msg"]; !hasMsg || msg != "info message" {
			t.Error("Expected JSON output to contain 'msg' field with 'info message'")
		}

		// Change to debug
		buf.Reset()
		log.SetLevel("debug")
		log.Debug("debug message")

		output = buf.String()
		logEntry = make(map[string]any)
		if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
			t.Errorf("Expected valid JSON output after level change, got: %v (output: %s)", err, output)
		}
		if msg, hasMsg := logEntry["msg"]; !hasMsg || msg != "debug message" {
			t.Error("Expected JSON output to contain 'msg' field with 'debug message'")
		}
		if level, hasLevel := logEntry["level"]; !hasLevel || level != "DEBUG" {
			t.Error("Expected JSON output to contain 'level' field with 'DEBUG'")
		}
	})
}
