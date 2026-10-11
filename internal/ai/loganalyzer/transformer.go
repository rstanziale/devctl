package loganalyzer

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Transformer converts the raw content of a log file into normalized LogEntry
// values. Each concrete transformer knows how to parse one file format.
type Transformer interface {
	// Transform parses data extracted from a file identified by source (the
	// file name without extension) into a slice of LogEntry values.
	Transform(source string, data []byte) ([]LogEntry, error)
}

// TransformerFor returns the Transformer able to handle the given file path,
// selected by its extension. The second return value is false when the format
// is not supported.
//
// OKD/structured logs are typically newline-delimited JSON saved with a ".log"
// extension (e.g. "r4c-gateway-27-7rrqs-r4c-gateway.log"), so ".log" is routed
// to the JSON transformer as well.
func TransformerFor(path string) (Transformer, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".har":
		return HARTransformer{}, true
	case ".json", ".log", ".ndjson":
		return JSONTransformer{}, true
	default:
		return nil, false
	}
}

// sourceName derives the log source name from a file path: the base name
// without its extension (e.g. "/log/test/client.har" -> "client").
func sourceName(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// parseTimestamp parses a timestamp string trying the layouts the analyzer
// understands, including the compact layouts and RFC3339 variants used by HAR.
func parseTimestamp(value string) (time.Time, error) {
	layouts := []string{
		LayoutCompactMillis,
		LayoutCompact,
		time.RFC3339Nano,
		time.RFC3339,
	}

	for _, layout := range layouts {
		if ts, err := time.Parse(layout, value); err == nil {
			return ts, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized timestamp %q", value)
}
