package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs-box/devctl/internal/config"
)

func TestNewRepoCloneCommand(t *testing.T) {
	repo := config.Repo{Name: "test", URL: "https://github.com/test/repo", Dir: "test/dir", Gitflow: true}
	workspace := "/tmp/workspace"
	cmd := NewRepoCloneCommand(repo, workspace)

	if cmd.repo != repo {
		t.Errorf("Expected repo %v, got %v", repo, cmd.repo)
	}
	if cmd.workspace != workspace {
		t.Errorf("Expected workspace %s, got %s", workspace, cmd.workspace)
	}
}

func TestName(t *testing.T) {
	repo := config.Repo{Name: "test-repo"}
	cmd := NewRepoCloneCommand(repo, "/tmp")
	expected := "Clone test-repo"
	if cmd.Name() != expected {
		t.Errorf("Expected name %s, got %s", expected, cmd.Name())
	}
}

func TestGetDestinationPath(t *testing.T) {
	repo := config.Repo{Dir: "backend/api"}
	workspace := "/home/user/projects"
	cmd := NewRepoCloneCommand(repo, workspace)
	expected := filepath.Join(workspace, repo.Dir)
	if cmd.getDestinationPath() != expected {
		t.Errorf("Expected path %s, got %s", expected, cmd.getDestinationPath())
	}
}

func TestDirectoryExists(t *testing.T) {
	// Create a temp dir
	tempDir := t.TempDir()
	cmd := NewRepoCloneCommand(config.Repo{}, "")

	// Test existing dir
	if !cmd.directoryExists(tempDir) {
		t.Error("Expected directory to exist")
	}

	// Test non-existing dir
	nonExistent := filepath.Join(tempDir, "nonexistent")
	if cmd.directoryExists(nonExistent) {
		t.Error("Expected directory to not exist")
	}
}

func TestCreateParentDirectories(t *testing.T) {
	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "a", "b", "c", "repo")
	cmd := NewRepoCloneCommand(config.Repo{}, "")

	err := cmd.createParentDirectories(dest)
	if err != nil {
		t.Errorf("Failed to create parent directories: %v", err)
	}

	// Check if parent dir exists
	parent := filepath.Dir(dest)
	if _, err := os.Stat(parent); os.IsNotExist(err) {
		t.Error("Parent directory was not created")
	}
}

func TestRunDirectoryExists(t *testing.T) {
	tempDir := t.TempDir()
	repo := config.Repo{Name: "test", URL: "https://github.com/octo-org/octo-template", Dir: "existing"}
	workspace := tempDir
	cmd := NewRepoCloneCommand(repo, workspace)

	// Create the final clone directory (dest/repoName)
	dest := cmd.getDestinationPath()
	cloneDest := filepath.Join(dest, "octo-template")
	os.MkdirAll(cloneDest, 0755)

	err := cmd.Run(context.Background())
	if err == nil || err.Error() != "directory "+cloneDest+" already exists" {
		t.Errorf("Expected error for existing directory, got %v", err)
	}
}

func TestGetRepoDirectoryName(t *testing.T) {
	tests := []struct {
		name     string
		repo     config.Repo
		expected string
	}{
		{
			name:     "https url",
			repo:     config.Repo{Name: "friendly", URL: "https://github.com/octo-org/octo-template.git"},
			expected: "octo-template",
		},
		{
			name:     "ssh scp style url",
			repo:     config.Repo{Name: "friendly", URL: "git@github.com:octo-org/octo-template.git"},
			expected: "octo-template",
		},
		{
			name:     "fallback to name",
			repo:     config.Repo{Name: "friendly", URL: ""},
			expected: "friendly",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewRepoCloneCommand(tc.repo, t.TempDir())
			if got := cmd.getRepoDirectoryName(); got != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, got)
			}
		})
	}
}

func TestExtractRepoNameFromURL(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{url: "https://github.com/octo-org/octo-template", expected: "octo-template"},
		{url: "https://github.com/octo-org/octo-template.git", expected: "octo-template"},
		{url: "git@github.com:octo-org/octo-template.git", expected: "octo-template"},
		{url: "", expected: ""},
	}

	for _, tc := range tests {
		if got := extractRepoNameFromURL(tc.url); got != tc.expected {
			t.Errorf("For url %q expected %q, got %q", tc.url, tc.expected, got)
		}
	}
}

// Note: For executeGitClone and executeGitFlowInit, integration tests would require git installed.
// For unit tests, we can mock exec.Command, but for simplicity, we'll skip detailed exec tests here.
// In a real scenario, use dependency injection for the exec function.
