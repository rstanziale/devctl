package loganalyzer

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveResponse(t *testing.T) {
	dir := t.TempDir()
	for _, response := range []string{"First analysis\nDetails: café\n", "Replacement analysis"} {
		before := time.Now().Format("20060102")
		path, err := SaveResponse(response, dir)
		after := time.Now().Format("20060102")
		if err != nil {
			t.Fatalf("save response: %v", err)
		}
		if filepath.Dir(path) != dir {
			t.Fatalf("response saved outside input directory: %q", path)
		}
		name := filepath.Base(path)
		if name != before+"_loganalyzer-response.txt" && name != after+"_loganalyzer-response.txt" {
			t.Fatalf("unexpected response filename: %q", name)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read response: %v", err)
		}
		if string(got) != response {
			t.Fatalf("saved response = %q, want %q", got, response)
		}
	}
}

func TestSaveResponsePreservesWriteError(t *testing.T) {
	path, err := SaveResponse("analysis", filepath.Join(t.TempDir(), "missing"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error = %v, want wrapped missing-directory error", err)
	}
	if path != "" {
		t.Fatalf("unexpected path on failure: %q", path)
	}
}

func mustParse(t *testing.T, value string) time.Time {
	t.Helper()
	ts, err := parseTimestamp(value)
	if err != nil {
		t.Fatalf("parse timestamp %q: %v", value, err)
	}
	return ts
}

func TestTransformerForSelectsByExtension(t *testing.T) {
	if _, ok := TransformerFor("client.har"); !ok {
		t.Fatal("expected HAR transformer for .har")
	}
	if tr, _ := TransformerFor("gateway.har"); tr != nil {
		if _, isHAR := tr.(HARTransformer); !isHAR {
			t.Fatalf("expected HARTransformer, got %T", tr)
		}
	}
	if tr, ok := TransformerFor("gateway.JSON"); !ok {
		t.Fatal("expected JSON transformer for .JSON (case-insensitive)")
	} else if _, isJSON := tr.(JSONTransformer); !isJSON {
		t.Fatalf("expected JSONTransformer, got %T", tr)
	}
	if tr, ok := TransformerFor("r4c-gateway-27-7rrqs-r4c-gateway.log"); !ok {
		t.Fatal("expected JSON transformer for .log")
	} else if _, isJSON := tr.(JSONTransformer); !isJSON {
		t.Fatalf("expected JSONTransformer for .log, got %T", tr)
	}
	if _, ok := TransformerFor("notes.txt"); ok {
		t.Fatal("expected unsupported extension to return false")
	}
}

func TestSourceNameStripsExtension(t *testing.T) {
	if got := sourceName(filepath.Join("log", "test", "client.har")); got != "client" {
		t.Fatalf("expected source 'client', got %q", got)
	}
}

func TestHARTransformerErrorAndWarn(t *testing.T) {
	har := `{
		"log": {
			"entries": [
				{
					"startedDateTime": "2025-04-14T11:20:00.013Z",
					"request": { "method": "GET", "url": "https://api/x" },
					"response": { "status": 500, "content": { "text": "" } }
				},
				{
					"startedDateTime": "2025-04-14T11:21:00.000Z",
					"request": { "method": "POST", "url": "https://api/y" },
					"response": {
						"status": 200,
						"content": { "text": "{\"state\":{\"code\":-1,\"description\":\"business warning\"}}" }
					}
				},
				{
					"startedDateTime": "2025-04-14T11:22:00.000Z",
					"request": { "method": "GET", "url": "https://api/z" },
					"response": {
						"status": 200,
						"content": { "text": "{\"state\":{\"code\":0,\"description\":\"ok\"}}" }
					}
				}
			]
		}
	}`

	entries, err := HARTransformer{}.Transform("client", []byte(har))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %+v", len(entries), entries)
	}

	if entries[0].Level != LevelError || entries[0].Source != "client" {
		t.Fatalf("expected ERROR client entry, got %+v", entries[0])
	}
	if !strings.Contains(entries[0].Message, "HTTP 500") {
		t.Fatalf("expected HTTP 500 in message, got %q", entries[0].Message)
	}

	if entries[1].Level != LevelWarn || entries[1].Message != "business warning" {
		t.Fatalf("expected WARN with description message, got %+v", entries[1])
	}
}

func TestJSONTransformerArrayAndNDJSON(t *testing.T) {
	array := `[
		{"timestamp": "20250414T11:20:00.013", "level": "error", "message": "boom"},
		{"time": "20250414T11:21:00.000", "severity": "warn", "msg": "careful"}
	]`

	entries, err := JSONTransformer{}.Transform("gateway", []byte(array))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Level != LevelError || entries[0].Message != "boom" {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}
	if entries[1].Level != LevelWarn || entries[1].Message != "careful" {
		t.Fatalf("unexpected second entry: %+v", entries[1])
	}

	ndjson := `{"@timestamp": "20250414T11:20:00.000", "level": "INFO", "message": "a"}
{"@timestamp": "20250414T11:21:00.000", "level": "ERROR", "message": "b"}`

	entries, err = JSONTransformer{}.Transform("server", []byte(ndjson))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 NDJSON entries, got %d", len(entries))
	}
}

func TestJSONTransformerOKDLogs(t *testing.T) {
	// Real samples extracted from an OKD pod (Vert.x / log4j-logstash output):
	// timestamp carries a timezone offset, "application" identifies the source.
	okd := `{"@timestamp":"2026-04-15T10:05:35.487+02:00","application":"r4c-gateway","message":"Vert.x logger initialized","location.file":"AbstractAppVerticle.java","level":"INFO","thread_name":"vert.x-eventloop-thread-1"}
{"@timestamp":"2026-04-15T10:05:36.001+02:00","application":"r4c-gateway","message":"connection refused","level":"ERROR","thread_name":"vert.x-eventloop-thread-1"}`

	entries, err := JSONTransformer{}.Transform("gateway-pod", []byte(okd))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %+v", len(entries), entries)
	}

	if entries[0].Source != "r4c-gateway" {
		t.Fatalf("expected source from application field, got %q", entries[0].Source)
	}
	if entries[0].Level != "INFO" {
		t.Fatalf("expected INFO level, got %q", entries[0].Level)
	}

	expectedTS := mustParse(t, "2026-04-15T10:05:35.487+02:00")
	if !entries[0].Timestamp.Equal(expectedTS) {
		t.Fatalf("timestamp with offset not parsed correctly: got %s", entries[0].Timestamp)
	}

	if entries[1].Level != LevelError || entries[1].Message != "connection refused" {
		t.Fatalf("unexpected error entry: %+v", entries[1])
	}
}

func TestJSONTransformerTolerantNDJSON(t *testing.T) {
	// An OKD .log file is not globally well-formed JSON: blank lines and
	// non-JSON noise (e.g. stack-trace continuations) may appear between
	// records. Valid records must still be extracted.
	noisy := `
{"@timestamp":"2026-04-15T10:05:35.487+02:00","application":"r4c-gateway","message":"starting","level":"INFO"}
	at eu.rs-box.xvalue.Foo.bar(Foo.java:42)
not even json
{"@timestamp":"2026-04-15T10:05:36.001+02:00","application":"r4c-gateway","message":"boom","level":"ERROR"}
{ broken json line
`

	entries, err := JSONTransformer{}.Transform("r4c-gateway-27-7rrqs-r4c-gateway", []byte(noisy))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 valid entries, got %d: %+v", len(entries), entries)
	}
	if entries[0].Message != "starting" || entries[1].Message != "boom" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
	if entries[1].Level != LevelError {
		t.Fatalf("expected ERROR, got %q", entries[1].Level)
	}
}

func TestFilterWindowLevelAndSort(t *testing.T) {
	entries := []LogEntry{
		{Timestamp: mustParse(t, "20250414T11:25:00.000"), Source: "a", Level: LevelError, Message: "late"},
		{Timestamp: mustParse(t, "20250414T11:21:00.000"), Source: "b", Level: LevelWarn, Message: "early"},
		{Timestamp: mustParse(t, "20250414T11:22:00.000"), Source: "c", Level: "INFO", Message: "dropped-level"},
		{Timestamp: mustParse(t, "20250414T11:40:00.000"), Source: "d", Level: LevelError, Message: "out-of-window"},
	}

	start := mustParse(t, "20250414T11:20:00")
	end := mustParse(t, "20250414T11:30:00")

	got := Filter(entries, start, end)
	if len(got) != 2 {
		t.Fatalf("expected 2 entries after filter, got %d: %+v", len(got), got)
	}
	if got[0].Message != "early" || got[1].Message != "late" {
		t.Fatalf("expected entries sorted ascending, got %+v", got)
	}
}

func TestBuildPromptInjectsEvents(t *testing.T) {
	entries := []LogEntry{
		{Timestamp: mustParse(t, "20250414T11:20:00.013"), Source: "client", Level: LevelError, Message: "An error occoured"},
	}

	prompt := BuildPrompt(entries)
	if strings.Contains(prompt, "{{events}}") {
		t.Fatal("placeholder was not replaced")
	}
	if !strings.Contains(prompt, "20250414T11:20:00.013 client ERROR An error occoured") {
		t.Fatalf("event line missing from prompt:\n%s", prompt)
	}
	if !strings.HasPrefix(prompt, "You are a senior SRE.") {
		t.Fatal("prompt does not start with the expected template")
	}
}

func TestSavePromptWritesFile(t *testing.T) {
	dir := t.TempDir()
	prompt := "You are a senior SRE.\n\nEvents:\n20260624T10:00:00.000 svc ERROR boom"

	path, err := SavePrompt(prompt, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Filename must match the pattern YYYYMMDD_loganalyzer-prompt.txt.
	base := filepath.Base(path)
	if len(base) != len("20060102_loganalyzer-prompt.txt") {
		t.Fatalf("unexpected filename length: %q", base)
	}
	if base[8:] != "_loganalyzer-prompt.txt" {
		t.Fatalf("unexpected filename suffix: %q", base)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read saved file: %v", err)
	}
	if string(data) != prompt {
		t.Fatalf("file content mismatch: got %q", string(data))
	}
}

func TestCollectScansSupportedFiles(t *testing.T) {
	dir := t.TempDir()

	har := `{"log":{"entries":[{"startedDateTime":"2025-04-14T11:20:00.000Z","request":{"method":"GET","url":"https://api/x"},"response":{"status":503,"content":{"text":""}}}]}}`
	if err := os.WriteFile(filepath.Join(dir, "client.har"), []byte(har), 0o600); err != nil {
		t.Fatal(err)
	}

	jsonLog := `[{"timestamp":"20250414T11:21:00.000","level":"ERROR","message":"db down"}]`
	if err := os.WriteFile(filepath.Join(dir, "gateway.json"), []byte(jsonLog), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("noise"), 0o600); err != nil {
		t.Fatal(err)
	}

	entries, err := Collect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries from supported files, got %d: %+v", len(entries), entries)
	}
}
