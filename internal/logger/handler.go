package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

const timestampLayout string = "2006-01-02T15:04:05.000"

type handler struct {
	level  slog.Leveler
	attrs  []slog.Attr
	groups []string
	mu     *sync.Mutex
}

func NewCustomHandler(level slog.Leveler) slog.Handler {
	return &handler{
		level: level,
		mu:    &sync.Mutex{},
	}
}

func (h *handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *handler) Handle(_ context.Context, record slog.Record) error {
	component := "General"
	fields := make([]string, 0, 8)

	appendAttr := func(attr slog.Attr) {
		h.collectAttr(attr, "", &component, &fields)
	}

	for _, attr := range h.attrs {
		appendAttr(attr)
	}

	record.Attrs(func(attr slog.Attr) bool {
		appendAttr(attr)
		return true
	})

	ts := record.Time
	if ts.IsZero() {
		ts = time.Now()
	}

	var b strings.Builder
	b.WriteString("[")
	b.WriteString(levelLabel(record.Level))
	b.WriteString("] ")
	b.WriteString(ts.Format(timestampLayout))
	b.WriteString(" [")
	b.WriteString(component)
	b.WriteString("] ")
	b.WriteString(record.Message)

	for _, field := range fields {
		b.WriteString(" ")
		b.WriteString(field)
	}
	b.WriteString("\n")

	stdout, stderr := currentWriters()
	writer := io.Writer(stdout)
	if record.Level >= slog.LevelError {
		writer = stderr
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(writer, b.String())
	return err
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cloned := *h
	cloned.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &cloned
}

func (h *handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	cloned := *h
	cloned.groups = append(append([]string{}, h.groups...), name)
	return &cloned
}

func (h *handler) collectAttr(attr slog.Attr, prefix string, component *string, fields *[]string) {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return
	}

	key := attr.Key
	if prefix != "" {
		key = prefix + "." + key
	}

	if key == "component" {
		*component = stringifyValue(attr.Value)
		return
	}

	if attr.Value.Kind() == slog.KindGroup {
		nextPrefix := key
		if key == "" {
			nextPrefix = prefix
		}
		for _, nested := range attr.Value.Group() {
			h.collectAttr(nested, nextPrefix, component, fields)
		}
		return
	}

	if key == "" {
		return
	}

	*fields = append(*fields, key+"="+stringifyValue(attr.Value))
}

func levelLabel(level slog.Level) string {
	switch {
	case level <= LevelTrace:
		return "TRACE"
	case level < slog.LevelInfo:
		return "DEBUG"
	case level < slog.LevelWarn:
		return "INFO"
	case level < slog.LevelError:
		return "WARN"
	default:
		return "ERROR"
	}
}

func stringifyValue(v slog.Value) string {
	switch v.Kind() {
	case slog.KindString:
		return quoteIfNeeded(v.String())
	case slog.KindInt64:
		return strconv.FormatInt(v.Int64(), 10)
	case slog.KindUint64:
		return strconv.FormatUint(v.Uint64(), 10)
	case slog.KindFloat64:
		return strconv.FormatFloat(v.Float64(), 'f', -1, 64)
	case slog.KindBool:
		return strconv.FormatBool(v.Bool())
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindTime:
		return v.Time().Format(time.RFC3339Nano)
	case slog.KindAny:
		return quoteIfNeeded(fmt.Sprint(v.Any()))
	default:
		return quoteIfNeeded(v.String())
	}
}

func quoteIfNeeded(value string) string {
	if value == "" {
		return "\"\""
	}
	if strings.ContainsAny(value, " \t\n\r=\"") {
		return strconv.Quote(value)
	}
	return value
}
