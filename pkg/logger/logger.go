package logger

import (
	"log"
	"os"
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
		stdLogger: log.New(os.Stdout, "", log.Ldate|log.Ltime),
	}
}

func (l *consoleLogger) Debug(msg string, args ...any) {
	l.stdLogger.Printf("[DEBUG] "+msg, args...)
}

func (l *consoleLogger) Info(msg string, args ...any) {
	l.stdLogger.Printf("[INFO] "+msg, args...)
}

func (l *consoleLogger) Warn(msg string, args ...any) {
	l.stdLogger.Printf("[WARN] "+msg, args...)
}

func (l *consoleLogger) Error(msg string, err error, args ...any) {
	if err != nil {
		l.stdLogger.Printf("[ERROR] "+msg+": %v", append(args, err)...)
	} else {
		l.stdLogger.Printf("[ERROR] "+msg, args...)
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
