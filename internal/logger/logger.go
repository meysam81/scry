// Package logger provides a thin wrapper around zerolog so that other internal
// packages do not import zerolog directly.
package logger

import (
	"context"

	"github.com/rs/zerolog"
)

type contextKey struct{}

// Logger wraps zerolog.Logger.
type Logger struct{ zl zerolog.Logger }

// New creates a Logger from a zerolog.Logger.
func New(zl zerolog.Logger) Logger { return Logger{zl: zl} }

// Nop returns a disabled Logger that discards all output.
func Nop() Logger { return Logger{zl: zerolog.Nop()} }

// Warn starts a new message with warn level.
func (l Logger) Warn() *zerolog.Event { return l.zl.Warn() }

// Info starts a new message with info level.
func (l Logger) Info() *zerolog.Event { return l.zl.Info() }

// Debug starts a new message with debug level.
func (l Logger) Debug() *zerolog.Event { return l.zl.Debug() }

// Error starts a new message with error level.
func (l Logger) Error() *zerolog.Event { return l.zl.Error() }

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
