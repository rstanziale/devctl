package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

const LevelTrace slog.Level = slog.LevelDebug - 4

type Logger struct {
	inner *slog.Logger
}

type Option func(*Options)

type Options struct {
	component string
}

var (
	initOnce sync.Once

	levelVar = &slog.LevelVar{}

	configMu     sync.RWMutex
	stdoutWriter io.Writer = os.Stdout
	stderrWriter io.Writer = os.Stderr

	baseLogger *slog.Logger
)

// Init configures the global logger verbosity.
// Verbosity mapping: 0 -> info, 1 -> debug, >=2 -> trace.
func Init(verboseCount int) {
	ensureInitialized()
	levelVar.Set(levelFromVerbose(verboseCount))
}

// GetLogger returns a contextual logger.
func GetLogger(opts ...Option) *Logger {
	ensureInitialized()

	cfg := Options{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	l := baseLogger
	if cfg.component != "" {
		l = l.With("component", cfg.component)
	}

	return &Logger{inner: l}
}

func (l *Logger) Trace(msg string, attrs ...any) {
	l.inner.Log(context.Background(), LevelTrace, msg, attrs...)
}

func (l *Logger) Debug(msg string, attrs ...any) {
	l.inner.Debug(msg, attrs...)
}

func (l *Logger) Info(msg string, attrs ...any) {
	l.inner.Info(msg, attrs...)
}

func (l *Logger) Warn(msg string, attrs ...any) {
	l.inner.Warn(msg, attrs...)
}

func (l *Logger) Error(msg string, attrs ...any) {
	l.inner.Error(msg, attrs...)
}

func WithComponent(name string) Option {
	return func(cfg *Options) {
		cfg.component = name
	}
}

func ensureInitialized() {
	initOnce.Do(func() {
		levelVar.Set(slog.LevelInfo)
		baseLogger = slog.New(NewCustomHandler(levelVar))
	})
}

func levelFromVerbose(verboseCount int) slog.Level {
	if verboseCount <= 0 {
		return slog.LevelInfo
	}
	if verboseCount == 1 {
		return slog.LevelDebug
	}
	return LevelTrace
}

// SetOutput redirects the logger's stdout/stderr writers. Passing nil for
// either argument restores the corresponding default (os.Stdout/os.Stderr).
// This exists to let tests in this and other internal packages capture log
// output deterministically; it is not intended for production use.
func SetOutput(stdout io.Writer, stderr io.Writer) {
	configMu.Lock()
	defer configMu.Unlock()

	if stdout != nil {
		stdoutWriter = stdout
	} else {
		stdoutWriter = os.Stdout
	}
	if stderr != nil {
		stderrWriter = stderr
	} else {
		stderrWriter = os.Stderr
	}
}

func currentWriters() (io.Writer, io.Writer) {
	configMu.RLock()
	defer configMu.RUnlock()
	return stdoutWriter, stderrWriter
}
