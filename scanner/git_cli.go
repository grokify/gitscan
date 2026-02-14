package scanner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CLIGitBackend implements GitBackend using git CLI commands.
// This is the fallback when go-git has issues or for compatibility.
type CLIGitBackend struct{}

// NewCLIGitBackend creates a new CLI git backend.
func NewCLIGitBackend() *CLIGitBackend {
	return &CLIGitBackend{}
}

// IsRepo checks if the path is a git repository by looking for .git directory.
func (c *CLIGitBackend) IsRepo(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// GetStatus uses `git status --porcelain -b` to check both uncommitted changes and unpushed commits.
// Output format:
//   - First line: ## branch...upstream [ahead N, behind M]
//   - Remaining lines: file status (if any uncommitted changes)
func (c *CLIGitBackend) GetStatus(repoPath string, checkUnpushed bool) (hasUncommitted, hasUnpushed bool) {
	cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain", "-b")
	output, err := cmd.Output()
	if err != nil {
		return false, false
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) == 0 {
		return false, false
	}

	// First line is branch info: ## main...origin/main [ahead 1]
	branchLine := lines[0]

	// Check for uncommitted changes (any non-empty lines after the first)
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) != "" {
			hasUncommitted = true
			break
		}
	}

	// Check for unpushed commits if requested
	if checkUnpushed {
		// Look for [ahead N] in the branch line
		if strings.Contains(branchLine, "[ahead") {
			hasUnpushed = true
		} else if !strings.Contains(branchLine, "...") {
			// No upstream configured (line is just "## main"), consider as unpushed
			hasUnpushed = true
		}
	}

	return hasUncommitted, hasUnpushed
}
