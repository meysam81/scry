// Package logger provides a thin wrapper around zerolog so that other internal
// packages do not import zerolog directly.
package logger

import (
	"context"
	"os"

	"github.com/rs/zerolog"
)

type contextKey struct{}

// Logger wraps zerolog.Logger.
type Logger struct{ zl zerolog.Logger }

// New creates a Logger from a zerolog.Logger.
func New(zl zerolog.Logger) Logger { return Logger{zl: zl} }

// Nop returns a disabled Logger that discards all output.
func Nop() Logger { return Logger{zl: zerolog.Nop()} }

// Setup creates a configured Logger based on level and format strings.
// Supported levels: debug, info, warn, error. Supported formats: json, pretty.
func Setup(level, format string) Logger {
	var zl zerolog.Logger

	switch format {
	case "json":
		zl = zerolog.New(os.Stderr).With().Timestamp().Logger()
	default:
		zl = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	}

	switch level {
	case "debug":
		zl = zl.Level(zerolog.DebugLevel)
	case "warn":
		zl = zl.Level(zerolog.WarnLevel)
	case "error":
		zl = zl.Level(zerolog.ErrorLevel)
	default:
		zl = zl.Level(zerolog.InfoLevel)
	}

	return Logger{zl: zl}
}

// Warn starts a new message with warn level.
func (l Logger) Warn() *zerolog.Event { return l.zl.Warn() }

// Info starts a new message with info level.
func (l Logger) Info() *zerolog.Event { return l.zl.Info() }

// Debug starts a new message with debug level.
func (l Logger) Debug() *zerolog.Event { return l.zl.Debug() }

// Error starts a new message with error level.
func (l Logger) Error() *zerolog.Event { return l.zl.Error() }

// Fatal starts a new message with fatal level.
// The message will call os.Exit(1) after being sent.
func (l Logger) Fatal() *zerolog.Event { return l.zl.Fatal() }

// WithContext returns a new context that carries l.
func WithContext(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext extracts the Logger from ctx, returning Nop() if absent.
func FromContext(ctx context.Context) Logger {
	if l, ok := ctx.Value(contextKey{}).(Logger); ok {
		return l
	}
	return Nop()
}
