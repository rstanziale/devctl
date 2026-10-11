package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDiscoverNonExistentDir(t *testing.T) {
	plugins, err := Discover("/non/existent/devctl/plugins")
	if err != nil {
		t.Fatalf("expected no error for non-existent dir, got: %v", err)
	}
	if plugins != nil {
		t.Fatalf("expected nil for non-existent dir, got %v", plugins)
	}
}

func TestDiscoverEmptyDir(t *testing.T) {
	dir := t.TempDir()
	plugins, err := Discover(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestDiscoverValidPlugin(t *testing.T) {
	dir := t.TempDir()
	filename, perm := platformBinary("devctl-myplugin")
	writeFile(t, filepath.Join(dir, filename), perm)

	plugins, err := Discover(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Name != "myplugin" {
		t.Errorf("expected name 'myplugin', got %q", plugins[0].Name)
	}
}

func TestDiscoverMultiWordPlugin(t *testing.T) {
	dir := t.TempDir()
	filename, perm := platformBinary("devctl-init-project")
	writeFile(t, filepath.Join(dir, filename), perm)

	plugins, err := Discover(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Name != "init-project" {
		t.Errorf("expected name 'init-project', got %q", plugins[0].Name)
	}
}

func TestDiscoverIgnoresNonPrefixedBinary(t *testing.T) {
	dir := t.TempDir()
	filename, perm := platformBinary("other-tool")
	writeFile(t, filepath.Join(dir, filename), perm)

	plugins, err := Discover(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestDiscoverIgnoresNonExecutableOnLinux(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit check not applicable on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "devctl-myplugin")
	writeFile(t, path, 0o644) // not executable

	plugins, err := Discover(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 0 {
		t.Fatalf("expected 0 plugins for non-executable file, got %d", len(plugins))
	}
}

func TestDiscoverIgnoresNonExeOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("applies to Windows only")
	}
	dir := t.TempDir()
	// File without .exe extension should be ignored on Windows
	writeFile(t, filepath.Join(dir, "devctl-myplugin"), 0o755)

	plugins, err := Discover(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 0 {
		t.Fatalf("expected 0 plugins (no .exe on Windows), got %d", len(plugins))
	}
}

func TestDiscoverIgnoresDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "devctl-subdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	plugins, err := Discover(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestPluginName(t *testing.T) {
	type testCase struct {
		input    string
		wantName string
		wantOK   bool
	}

	var tests []testCase
	if runtime.GOOS == "windows" {
		tests = []testCase{
			{"devctl-myplugin.exe", "myplugin", true},
			{"devctl-init-project.exe", "init-project", true},
			{"devctl-myplugin", "", false}, // missing .exe on Windows
			{"devctl-.exe", "", false},     // empty name
			{"devctl.exe", "", false},      // no separator
			{"other-tool.exe", "", false},
		}
	} else {
		tests = []testCase{
			{"devctl-myplugin", "myplugin", true},
			{"devctl-init-project", "init-project", true},
			{"devctl-", "", false}, // empty name
			{"devctl", "", false},  // no separator
			{"other-tool", "", false},
		}
	}

	for _, tt := range tests {
		name, ok := pluginName(tt.input)
		if ok != tt.wantOK {
			t.Errorf("pluginName(%q): ok=%v, want %v", tt.input, ok, tt.wantOK)
		}
		if name != tt.wantName {
			t.Errorf("pluginName(%q): name=%q, want %q", tt.input, name, tt.wantName)
		}
	}
}

// platformBinary returns the filename and permissions for a plugin binary on the current OS.
func platformBinary(base string) (filename string, perm os.FileMode) {
	if runtime.GOOS == "windows" {
		return base + ".exe", 0o644
	}
	return base, 0o755
}

func writeFile(t *testing.T, path string, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho hello"), perm); err != nil {
		t.Fatal(err)
	}
}
