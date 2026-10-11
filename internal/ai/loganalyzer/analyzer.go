package loganalyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs-box/devctl/internal/logger"
)

// promptTemplate is sent to the AI engine. The "{{sources}}" placeholder is
// replaced with the list of log sources found in the events, and "{{events}}"
// with the time-ordered events themselves.
const promptTemplate = `You are a senior SRE.

You are given logs from multiple sources:
{{sources}}

The logs are already time-ordered.

Tasks:
1. Identify correlated failures across systems
2. Reconstruct user-impacting issues
3. Identify root causes
4. Suggest fixes

Events:
{{events}}`

// Collect scans dir for supported log files, transforms each of them through the
// appropriate Transformer and returns the aggregated log entries. Files with an
// unsupported format are ignored. Errors affecting a single file are logged and
// skipped so that one malformed file does not abort the whole analysis.
func Collect(dir string) ([]LogEntry, error) {
	log := logger.GetLogger(logger.WithComponent("LogAnalyzerCollect"))

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", dir, err)
	}

	var entries []LogEntry
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}

		path := filepath.Join(dir, de.Name())
		transformer, ok := TransformerFor(path)
		if !ok {
			log.Debug("skipping unsupported file", "file", de.Name())
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			log.Error("failed to read file", "file", de.Name(), "error", err)
			continue
		}

		transformed, err := transformer.Transform(sourceName(path), data)
		if err != nil {
			log.Error("failed to transform file", "file", de.Name(), "error", err)
			continue
		}

		log.Debug("transformed file", "file", de.Name(), "entries", len(transformed))
		entries = append(entries, transformed...)
	}

	return entries, nil
}

// Filter keeps only WARN and ERROR entries whose timestamp falls within the
// inclusive [start, end] window and returns them sorted by timestamp ascending.
func Filter(entries []LogEntry, start, end time.Time) []LogEntry {
	filtered := make([]LogEntry, 0, len(entries))
	for _, e := range entries {
		if e.Level != LevelWarn && e.Level != LevelError {
			continue
		}
		if e.Timestamp.Before(start) || e.Timestamp.After(end) {
			continue
		}
		filtered = append(filtered, e)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.Before(filtered[j].Timestamp)
	})

	return filtered
}

// SavePrompt writes prompt to a file named "<YYYYMMDD>_loganalyzer-prompt.txt" inside
// dir and returns the full path. It is the fallback used when the AI engine is
// unavailable so that the caller can run the analysis manually later.
func SavePrompt(prompt, dir string) (string, error) {
	return saveArtifact(prompt, dir, "loganalyzer-prompt.txt")
}

// SaveResponse persists a completed analysis alongside its source logs for later review.
// Repeated analyses on the same day replace the previous response.
func SaveResponse(response, dir string) (string, error) {
	return saveArtifact(response, dir, "loganalyzer-response.txt")
}

// saveArtifact writes content to a file named "<YYYYMMDD>_<suffix>" inside dir
// and returns the full path.
func saveArtifact(content, dir, suffix string) (string, error) {
	filename := time.Now().Format("20060102") + "_" + suffix
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("save %s to %q: %w", suffix, path, err)
	}
	return path, nil
}

// BuildPrompt renders the analysis prompt by injecting the distinct log
// sources and the time-ordered events into the prompt template.
func BuildPrompt(entries []LogEntry) string {
	var events strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&events, "%s %s %s %s\n",
			e.Timestamp.Format(LayoutCompactMillis),
			e.Source,
			e.Level,
			e.Message,
		)
	}

	prompt := strings.ReplaceAll(promptTemplate, "{{sources}}", sourcesList(entries))
	eventsStr := strings.TrimRight(events.String(), "\n")
	return strings.ReplaceAll(prompt, "{{events}}", eventsStr)
}

// sourcesList returns a "- name" bullet list of the distinct sources present
// in entries, in first-seen order.
func sourcesList(entries []LogEntry) string {
	seen := make(map[string]struct{})
	var b strings.Builder
	for _, e := range entries {
		if _, ok := seen[e.Source]; ok {
			continue
		}
		seen[e.Source] = struct{}{}
		fmt.Fprintf(&b, "- %s\n", e.Source)
	}
	return strings.TrimRight(b.String(), "\n")
}
