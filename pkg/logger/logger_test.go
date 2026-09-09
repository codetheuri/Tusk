package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"strings"
	"testing"
)

func TestConsoleLogger(t *testing.T) {
	var buf bytes.Buffer
	l := &consoleLogger{
		stdLogger: log.New(&buf, "", 0),
	}

	l.Info("user logged in", "user_id", 42, "role", "admin")
	output := buf.String()

	if !strings.Contains(output, "[INFO] user logged in user_id=42, role=admin") {
		t.Errorf("unexpected log output: %s", output)
	}
}

func TestConsoleLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	l := &consoleLogger{
		stdLogger: log.New(&buf, "", 0),
	}

	l.Error("db error", errors.New("test panicked"), "query", "SELECT 1")
	output := buf.String()

	if !strings.Contains(output, "[ERROR] db error: test panicked query=SELECT 1") {
		t.Errorf("unexpected error log output: %s", output)
	}
}

func TestSlogLogger_JSONIsQueryable(t *testing.T) {
	var buf bytes.Buffer
	l := &slogLogger{l: slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))}

	l.Info("user logged in", "user_id", 42, "role", "admin")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("output was not valid JSON: %v (%q)", err, buf.String())
	}
	if record["msg"] != "user logged in" {
		t.Errorf("msg = %v, want %q", record["msg"], "user logged in")
	}
	// The point of structured logging: user_id is its own field, not text
	// embedded in the message that would need a regex to find.
	if record["user_id"] != float64(42) {
		t.Errorf("user_id = %v, want 42", record["user_id"])
	}
	if record["role"] != "admin" {
		t.Errorf("role = %v, want admin", record["role"])
	}
}

func TestSlogLogger_ErrorAttachesErrorField(t *testing.T) {
	var buf bytes.Buffer
	l := &slogLogger{l: slog.New(slog.NewJSONHandler(&buf, nil))}

	l.Error("save failed", errors.New("connection refused"), "table", "users")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("output was not valid JSON: %v", err)
	}
	if record["error"] != "connection refused" {
		t.Errorf("error field = %v, want %q", record["error"], "connection refused")
	}
	if record["table"] != "users" {
		t.Errorf("table = %v, want users", record["table"])
	}
}

func TestSlogLogger_RespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	l := &slogLogger{l: slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))}

	l.Info("this should be filtered out")
	if buf.Len() != 0 {
		t.Errorf("expected Info to be suppressed at Warn level, got: %s", buf.String())
	}

	l.Warn("this should appear")
	if buf.Len() == 0 {
		t.Error("expected Warn to be emitted at Warn level")
	}
}

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug":    slog.LevelDebug,
		"DEBUG":    slog.LevelDebug,
		"warn":     slog.LevelWarn,
		"warning":  slog.LevelWarn,
		"error":    slog.LevelError,
		"info":     slog.LevelInfo,
		"":         slog.LevelInfo,
		"nonsense": slog.LevelInfo, // must not silence logging
	}
	for input, want := range cases {
		if got := parseLevel(input); got != want {
			t.Errorf("parseLevel(%q) = %v, want %v", input, got, want)
		}
	}
}
