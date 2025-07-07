package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

// Logger interface for logging operations
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, err error, args ...any)
	Fatal(msg string, err error, args ...any)
}

// logs to console
type consoleLogger struct {
	stdLogger *log.Logger
}

// NewConsoleLogger creates a new console logger
func NewConsoleLogger() Logger {
	return &consoleLogger{
		stdLogger: log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmsgprefix),
	}
}

//format args
func formatArgs(args ...any) string {
	if len(args) == 0 {
		return ""
	}
	var parts []string
	for i := 0; i< len(args); i += 2 {
		key := fmt.Sprintf("%v", args[i])
		if i+1 < len(args) {
			value := fmt.Sprintf("%v", args[i+1])
			parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}else {
		 parts = append(parts, fmt.Sprintf("%s=<no-value>",key))
	}
	}
	return " " + strings.Join(parts, ", ")
}
func (l *consoleLogger) Debug(msg string, args ...any) {
	l.stdLogger.Printf("[DEBUG] %s%s", msg, formatArgs(args...))
}

func (l *consoleLogger) Info(msg string, args ...any) {
	l.stdLogger.Printf("[INFO] %s%s", msg, formatArgs(args...))

}

func (l *consoleLogger) Warn(msg string, args ...any) {
	l.stdLogger.Printf("[WARN] %s%s", msg, formatArgs(args...))
}

func (l *consoleLogger) Error(msg string, err error, args ...any) {
	formatArgs := formatArgs(args...)
	if err != nil {
		l.stdLogger.Printf("[ERROR] %s: %v%s", msg, formatArgs, err)
	} else {
		l.stdLogger.Printf("[ERROR] %s%s", msg, formatArgs)
	}
}

func (l *consoleLogger) Fatal(msg string, err error, args ...any) {
	l.Error(msg, err, args...)
	os.Exit(1)
}

// global logger instance
var (
	globalLogger Logger = NewConsoleLogger()
	loggerMuteex sync.RWMutex
)

// GetLogger returns the global logger instance
func GetLogger() Logger {
	loggerMuteex.RLock()
	defer loggerMuteex.RUnlock()
	return globalLogger
}

func SetGlobalLogger(l Logger) {
	loggerMuteex.Lock()
	defer loggerMuteex.Unlock()
	globalLogger = l
}
