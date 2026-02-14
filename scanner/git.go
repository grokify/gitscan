package scanner

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// GitBackend provides git operations for repository scanning.
type GitBackend interface {
	// IsRepo checks if the path is a git repository.
	IsRepo(path string) bool
	// GetStatus returns uncommitted changes and unpushed commits status.
	GetStatus(repoPath string, checkUnpushed bool) (hasUncommitted, hasUnpushed bool)
}

// GoGitBackend implements GitBackend using go-git (pure Go, no process spawning).
type GoGitBackend struct{}

// NewGoGitBackend creates a new go-git backend.
func NewGoGitBackend() *GoGitBackend {
	return &GoGitBackend{}
}

// IsRepo checks if the path is a git repository using go-git.
func (g *GoGitBackend) IsRepo(path string) bool {
	_, err := git.PlainOpen(path)
	return err == nil
}

// GetStatus returns uncommitted changes and unpushed commits status using go-git.
func (g *GoGitBackend) GetStatus(repoPath string, checkUnpushed bool) (hasUncommitted, hasUnpushed bool) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return false, false
	}

	// Check for uncommitted changes
	worktree, err := repo.Worktree()
	if err != nil {
		return false, false
	}

	status, err := worktree.Status()
	if err != nil {
		return false, false
	}

	hasUncommitted = !status.IsClean()

	// Check for unpushed commits if requested
	if checkUnpushed {
		hasUnpushed = g.hasUnpushedCommits(repo)
	}

	return hasUncommitted, hasUnpushed
}

// hasUnpushedCommits checks if HEAD is ahead of its upstream tracking branch.
func (g *GoGitBackend) hasUnpushedCommits(repo *git.Repository) bool {
	// Get HEAD reference
	head, err := repo.Head()
	if err != nil {
		return true // No HEAD, consider as unpushed
	}

	// Get the current branch name
	if !head.Name().IsBranch() {
		return false // Detached HEAD, skip unpushed check
	}

	branchName := head.Name().Short()

	// Try to find the remote tracking branch
	// Convention: origin/<branch>
	remoteRef, err := repo.Reference(plumbing.NewRemoteReferenceName("origin", branchName), true)
	if err != nil {
		// No remote tracking branch, consider as unpushed
		return true
	}

	// Compare HEAD with remote
	headCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return false
	}

	remoteCommit, err := repo.CommitObject(remoteRef.Hash())
	if err != nil {
		return true // Remote ref exists but can't get commit, assume unpushed
	}

	// If HEAD and remote point to same commit, nothing to push
	if head.Hash() == remoteRef.Hash() {
		return false
	}

	// Check if remote commit is ancestor of HEAD (we're ahead)
	isAncestor, err := headCommit.IsAncestor(remoteCommit)
	if err != nil {
		return true // Error checking, assume unpushed
	}

	// If remote is ancestor of HEAD, we have unpushed commits
	return isAncestor
}

// DefaultGitBackend returns the default git backend (go-git).
func DefaultGitBackend() GitBackend {
	return NewGoGitBackend()
}
