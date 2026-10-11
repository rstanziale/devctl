package loganalyzer

import "time"

// Log levels handled by the analyzer.
const (
	LevelWarn  = "WARN"
	LevelError = "ERROR"
)

// Timestamp layouts used across the analyzer.
const (
	// LayoutCompactMillis matches timestamps such as "20250414T11:20:00.013".
	LayoutCompactMillis = "20060102T15:04:05.000"
	// LayoutCompact matches timestamps such as "20250414T11:20:00".
	LayoutCompact = "20060102T15:04:05"
)

// LogEntry is the normalized, manipulable representation of a single log event
// extracted from a source file.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}
