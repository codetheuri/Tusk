package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// slogLogger adapts log/slog to the Logger interface.
//
// slog has been in the standard library since Go 1.21 and is the idiomatic
// answer for structured logging — which is why this is an adapter rather than a
// dependency on a third-party logging package.
//
// The value of structured output is not prettiness: it is that every field is
// separately queryable. "user_id=42" inside a formatted string requires a regex
// to search; a JSON field does not. Once logs are shipped anywhere central, that
// difference decides whether an incident takes minutes or hours.
type slogLogger struct {
	l *slog.Logger
}

// NewJSONLogger writes one JSON object per line at the given level.
//
// This is the production form: machine-parseable, and stable enough that log
// processors can rely on the field names.
func NewJSONLogger(level string) Logger {
	return &slogLogger{l: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(level),
	}))}
}

// NewTextLogger writes human-readable key=value lines at the given level.
// Suited to a developer's terminal, where a wall of JSON is a hindrance.
func NewTextLogger(level string) Logger {
	return &slogLogger{l: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(level),
	}))}
}

// New selects a logger appropriate to the environment: JSON in production,
// readable text elsewhere.
//
// Choosing here rather than at each call site means no code has to know which
// environment it is running in — the only distinction that ever mattered was how
// the bytes are formatted.
func New(production bool, level string) Logger {
	if production {
		return NewJSONLogger(level)
	}
	return NewTextLogger(level)
}

// WithAttrs returns a logger that stamps the given key/value pairs onto every
// record. Use it to bind a request ID once rather than threading it through
// every call.
func (s *slogLogger) WithAttrs(args ...any) Logger {
	return &slogLogger{l: s.l.With(args...)}
}

func (s *slogLogger) Debug(msg string, args ...any) { s.l.Debug(msg, args...) }
func (s *slogLogger) Info(msg string, args ...any)  { s.l.Info(msg, args...) }
func (s *slogLogger) Warn(msg string, args ...any)  { s.l.Warn(msg, args...) }

func (s *slogLogger) Error(msg string, err error, args ...any) {
	if err != nil {
		args = append(args, "error", err.Error())
	}
	s.l.Error(msg, args...)
}

// Fatal logs at error level and terminates the process.
//
// Reserved for unrecoverable startup failures — a process that cannot reach its
// database or load its configuration should stop loudly rather than serve
// requests it cannot fulfil. Never call it from request handling.
func (s *slogLogger) Fatal(msg string, err error, args ...any) {
	s.Error(msg, err, args...)
	os.Exit(1)
}

// LogAttrs is available for hot paths where allocation matters; the variadic
// args form allocates a slice per call.
func (s *slogLogger) LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	s.l.LogAttrs(ctx, level, msg, attrs...)
}

// parseLevel maps a configured level name onto a slog level, defaulting to Info
// for anything unrecognised. An unreadable LOG_LEVEL should not silence logging.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
