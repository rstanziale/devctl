package bootstrap

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs-box/devctl/internal/config"
	"github.com/rs-box/devctl/internal/logger"
)

// RepoCloneCommand implements the Command interface for cloning a repository
type RepoCloneCommand struct {
	repo      config.Repo
	workspace string
}

var log = logger.GetLogger(logger.WithComponent("RepoCloneCommand"))

// NewRepoCloneCommand creates a new RepoCloneCommand
func NewRepoCloneCommand(repo config.Repo, workspace string) *RepoCloneCommand {
	return &RepoCloneCommand{
		repo:      repo,
		workspace: workspace,
	}
}

// Name returns the name of the command
func (c *RepoCloneCommand) Name() string {
	return fmt.Sprintf("Clone %s", c.repo.Name)
}

// Run executes the git clone and optionally git flow init
func (c *RepoCloneCommand) Run(ctx context.Context) error {
	start := time.Now()

	dest := c.getDestinationPath()
	cloneDest := filepath.Join(dest, c.getRepoDirectoryName())

	log.Debug("starting bootstrap", "repo", c.repo.Name, "destination", cloneDest)
	log.Debug("source url", "repo", c.repo.Name, "url", c.repo.URL)

	if c.directoryExists(cloneDest) {
		return fmt.Errorf("directory %s already exists", cloneDest)
	}

	if err := c.createParentDirectories(cloneDest); err != nil {
		return err
	}

	cloneStart := time.Now()
	log.Debug("running git clone", "repo", c.repo.Name, "url", c.repo.URL, "destination", cloneDest)
	if err := c.executeGitClone(ctx, c.repo.URL, cloneDest); err != nil {
		return err
	}
	log.Debug("clone completed", "repo", c.repo.Name, "elapsed", time.Since(cloneStart).Round(time.Millisecond))

	if c.repo.Gitflow {
		flowStart := time.Now()
		log.Debug("running git flow init", "repo", c.repo.Name)
		if err := c.executeGitFlowInit(ctx, cloneDest); err != nil {
			return err
		}
		log.Debug("gitflow initialized", "repo", c.repo.Name, "elapsed", time.Since(flowStart).Round(time.Millisecond))
	} else {
		log.Debug("gitflow disabled", "repo", c.repo.Name)
	}

	log.Debug("repository bootstrap completed", "repo", c.repo.Name, "elapsed", time.Since(start).Round(time.Millisecond))

	return nil
}

// getRepoDirectoryName returns the directory name for cloning.
// It prefers the repository URL tail (e.g. octo-template from .../octo-template.git)
// and falls back to the configured repo name when extraction is not possible.
func (c *RepoCloneCommand) getRepoDirectoryName() string {
	if name := extractRepoNameFromURL(c.repo.URL); name != "" {
		return name
	}

	return c.repo.Name
}

// getDestinationPath returns the full path where the repo should be cloned
func (c *RepoCloneCommand) getDestinationPath() string {
	return filepath.Join(c.workspace, c.repo.Dir)
}

// directoryExists checks if the directory already exists
func (c *RepoCloneCommand) directoryExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// createParentDirectories creates the parent directories for the destination path
func (c *RepoCloneCommand) createParentDirectories(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directories for %s: %w", path, err)
	}
	return nil
}

// extractRepoNameFromURL extracts the repository folder name from common git URL formats.
func extractRepoNameFromURL(repoURL string) string {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return ""
	}

	var repoPath string
	if strings.Contains(repoURL, "://") {
		parsedURL, err := url.Parse(repoURL)
		if err != nil {
			return ""
		}
		repoPath = parsedURL.Path
	} else {
		// Handle scp-like syntax, for example: git@github.com:org/repo.git
		colonIdx := strings.LastIndex(repoURL, ":")
		slashIdx := strings.LastIndex(repoURL, "/")
		if colonIdx > slashIdx {
			repoPath = repoURL[colonIdx+1:]
		} else {
			repoPath = repoURL
		}
	}

	name := path.Base(strings.Trim(repoPath, "/"))
	name = strings.TrimSuffix(name, ".git")
	if name == "." || name == "" {
		return ""
	}

	return name
}

// executeGitClone performs the git clone command
func (c *RepoCloneCommand) executeGitClone(ctx context.Context, url string, dest string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", url, dest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed for %s: %w (output: %s)", dest, err, string(output))
	}
	return nil
}

// executeGitFlowInit performs the git flow init command
func (c *RepoCloneCommand) executeGitFlowInit(ctx context.Context, dest string) error {
	cmd := exec.CommandContext(ctx, "git", "flow", "init", "-d")
	cmd.Dir = dest
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git flow init failed in %s: %w (output: %s)", dest, err, string(output))
	}
	return nil
}
