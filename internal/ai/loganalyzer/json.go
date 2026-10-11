package loganalyzer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// JSONTransformer turns structured JSON logs (e.g. the gateway or another OKD
// pod) into log entries. It accepts either a JSON array of log objects or
// newline-delimited JSON objects (NDJSON).
type JSONTransformer struct{}

// rawJSONLog tolerates the most common field-name variants found in structured
// logs (including OKD/Vert.x logstash output).
type rawJSONLog struct {
	Timestamp   string `json:"timestamp"`
	Time        string `json:"time"`
	AtTime      string `json:"@timestamp"`
	Level       string `json:"level"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Msg         string `json:"msg"`
	Application string `json:"application"`
	App         string `json:"app"`
	Service     string `json:"service"`
}

// Transform implements Transformer for structured JSON logs.
func (JSONTransformer) Transform(source string, data []byte) ([]LogEntry, error) {
	raws, err := decodeJSONLogs(data)
	if err != nil {
		return nil, err
	}

	entries := make([]LogEntry, 0, len(raws))
	for _, r := range raws {
		ts, err := parseTimestamp(firstNonEmpty(r.Timestamp, r.Time, r.AtTime))
		if err != nil {
			continue
		}

		entries = append(entries, LogEntry{
			Timestamp: ts,
			// Prefer the application/service name carried by the log so that
			// cross-system correlation is based on the real source rather than
			// the file name. Fall back to the file name when absent.
			Source:  firstNonEmpty(r.Application, r.App, r.Service, source),
			Level:   strings.ToUpper(firstNonEmpty(r.Level, r.Severity)),
			Message: firstNonEmpty(r.Message, r.Msg),
		})
	}

	return entries, nil
}

// decodeJSONLogs parses data as a JSON array first, falling back to
// newline-delimited JSON (NDJSON). NDJSON is parsed line by line and lines that
// are not valid JSON objects (blank lines, stack-trace continuations, plain
// text, ...) are skipped, so a file that is not globally well-formed JSON still
// yields its valid log records.
func decodeJSONLogs(data []byte) ([]rawJSONLog, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, nil
	}

	if trimmed[0] == '[' {
		var raws []rawJSONLog
		if err := json.Unmarshal(trimmed, &raws); err != nil {
			return nil, fmt.Errorf("parse JSON array: %w", err)
		}
		return raws, nil
	}

	var raws []rawJSONLog
	for _, line := range bytes.Split(trimmed, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] != '{' {
			continue
		}

		var r rawJSONLog
		if err := json.Unmarshal(line, &r); err != nil {
			// Tolerate malformed lines instead of failing the whole file.
			continue
		}
		raws = append(raws, r)
	}

	return raws, nil
}

// firstNonEmpty returns the first non-empty string from values.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
