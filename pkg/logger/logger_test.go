package logger

import (
	"bytes"
	"errors"
	"log"
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
