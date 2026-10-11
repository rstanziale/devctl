package loganalyzer

import (
	"encoding/json"
	"fmt"
)

// HARTransformer turns an HTTP Archive (HAR) file into log entries.
//
// A HAR file does not contain browser console logs: it captures outgoing HTTP
// calls and their responses. The transformer applies the following strategy:
//
//   - If the call failed (no response or an HTTP error status), an ERROR entry
//     is produced.
//   - If the response body contains a structure with "state.code" equal to -1,
//     a WARN entry is produced whose message comes from "state.description".
type HARTransformer struct{}

type harFile struct {
	Log harLog `json:"log"`
}

type harLog struct {
	Entries []harEntry `json:"entries"`
}

type harEntry struct {
	StartedDateTime string      `json:"startedDateTime"`
	Request         harRequest  `json:"request"`
	Response        harResponse `json:"response"`
}

type harRequest struct {
	Method string `json:"method"`
	URL    string `json:"url"`
}

type harResponse struct {
	Status  int        `json:"status"`
	Content harContent `json:"content"`
}

type harContent struct {
	Text string `json:"text"`
}

type harBody struct {
	State struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"state"`
}

// Transform implements Transformer for HAR files.
func (HARTransformer) Transform(source string, data []byte) ([]LogEntry, error) {
	var file harFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse HAR: %w", err)
	}

	entries := make([]LogEntry, 0, len(file.Log.Entries))
	for _, e := range file.Log.Entries {
		ts, err := parseTimestamp(e.StartedDateTime)
		if err != nil {
			// Skip entries with an unparsable timestamp rather than failing
			// the whole file.
			continue
		}

		if isHTTPError(e.Response.Status) {
			entries = append(entries, LogEntry{
				Timestamp: ts,
				Source:    source,
				Level:     LevelError,
				Message: fmt.Sprintf(
					"%s %s -> HTTP %d",
					e.Request.Method, e.Request.URL, e.Response.Status,
				),
			})
			continue
		}

		if desc, ok := warnFromBody(e.Response.Content.Text); ok {
			entries = append(entries, LogEntry{
				Timestamp: ts,
				Source:    source,
				Level:     LevelWarn,
				Message:   desc,
			})
		}
	}

	return entries, nil
}

// isHTTPError reports whether the response status denotes a failed call. A zero
// status indicates the call never received a response (network error).
func isHTTPError(status int) bool {
	return status == 0 || status >= 400
}

// warnFromBody inspects a response body and returns the description to use for a
// WARN entry when the body carries a "state.code" equal to -1.
func warnFromBody(text string) (string, bool) {
	if text == "" {
		return "", false
	}

	var body harBody
	if err := json.Unmarshal([]byte(text), &body); err != nil {
		return "", false
	}

	if body.State.Code != -1 {
		return "", false
	}

	return body.State.Description, true
}
