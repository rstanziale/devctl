package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}
	return path
}

func TestLoadValidConfig(t *testing.T) {
	path := writeConfigFile(t, `
version: 1
workspace: /tmp/workspace
`)

	cfg, usedPath, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.Workspace != "/tmp/workspace" {
		t.Errorf("expected workspace /tmp/workspace, got %q", cfg.Workspace)
	}
	if usedPath != path {
		t.Errorf("expected used path %q, got %q", path, usedPath)
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.yaml")

	_, _, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing config file, got nil")
	}
}

func TestLoadMalformedYAML(t *testing.T) {
	path := writeConfigFile(t, "version: [1, 2\nworkspace: broken")

	_, _, err := Load(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}
