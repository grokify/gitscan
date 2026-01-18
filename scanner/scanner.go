package scanner

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// GoModResult holds analysis results for a single go.mod file.
type GoModResult struct {
	Path         string   // Path to go.mod relative to repo root
	ModuleName   string   // Module name from go.mod
	Dependencies []string // Required module paths
	ReplaceCount int      // Number of replace directives
}

// RepoResult holds the analysis results for a single repository.
type RepoResult struct {
	Name                  string
	Path                  string
	IsGitRepo             bool
	HasGoMod              bool
	HasUncommittedChanges bool
	HasReplaceDirectives  bool
	HasModuleMismatch     bool
	ModuleName            string
	ReplaceCount          int
	Dependencies          []string      // Dependencies from root go.mod
	GoModFiles            []GoModResult // All go.mod files (when recurse=true)
}

// HasDependency checks if the repo depends on the given module path.
// When GoModFiles is populated (recurse mode), checks all go.mod files.
func (r RepoResult) HasDependency(modulePath string) bool {
	// Check root dependencies
	if slices.Contains(r.Dependencies, modulePath) {
		return true
	}
	// Check nested go.mod files
	for _, gm := range r.GoModFiles {
		if slices.Contains(gm.Dependencies, modulePath) {
			return true
		}
	}
	return false
}

// ProgressFunc is called during scanning with current progress.
type ProgressFunc func(current, total int, name string)

// ScanOptions configures the scanning behavior.
type ScanOptions struct {
	Recurse bool // Search for nested go.mod files
}

// CountDirectories counts the number of scannable directories.
func CountDirectories(dirPath string) (int, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		count++
	}
	return count, nil
}

// ScanDirectory scans all direct subdirectories in the given path.
func ScanDirectory(dirPath string) ([]RepoResult, error) {
	return ScanDirectoryWithProgress(dirPath, nil, ScanOptions{})
}

// ScanDirectoryWithProgress scans directories and reports progress via callback.
func ScanDirectoryWithProgress(dirPath string, progressFn ProgressFunc, opts ScanOptions) ([]RepoResult, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	// First pass: count directories
	var dirs []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		dirs = append(dirs, entry)
	}

	total := len(dirs)
	var results []RepoResult

	for i, entry := range dirs {
		if progressFn != nil {
			progressFn(i+1, total, entry.Name())
		}

		subPath := filepath.Join(dirPath, entry.Name())
		result := analyzeRepo(subPath, entry.Name(), opts)
		results = append(results, result)
	}

	return results, nil
}

func analyzeRepo(repoPath, name string, opts ScanOptions) RepoResult {
	result := RepoResult{
		Name: name,
		Path: repoPath,
	}

	// Check if it's a git repository
	result.IsGitRepo = isGitRepo(repoPath)

	// Check for uncommitted changes
	if result.IsGitRepo {
		result.HasUncommittedChanges = hasUncommittedChanges(repoPath)
	}

	// Analyze go.mod at root
	goModPath := filepath.Join(repoPath, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		result.HasGoMod = true
		moduleName, replaceCount, dependencies := analyzeGoMod(goModPath)
		result.ModuleName = moduleName
		result.ReplaceCount = replaceCount
		result.HasReplaceDirectives = replaceCount > 0
		result.Dependencies = dependencies

		// Check if module name matches directory structure
		if moduleName != "" {
			result.HasModuleMismatch = !moduleMatchesPath(moduleName, name)
		}
	}

	// Find nested go.mod files if recurse is enabled
	if opts.Recurse {
		goModFiles := findGoModFiles(repoPath)
		for _, goModFile := range goModFiles {
			relPath, _ := filepath.Rel(repoPath, goModFile)
			moduleName, replaceCount, dependencies := analyzeGoMod(goModFile)
			result.GoModFiles = append(result.GoModFiles, GoModResult{
				Path:         relPath,
				ModuleName:   moduleName,
				Dependencies: dependencies,
				ReplaceCount: replaceCount,
			})
		}
	}

	return result
}

// findGoModFiles recursively finds all go.mod files in the given directory.
// Skips vendor directories and hidden directories.
func findGoModFiles(rootPath string) []string {
	var goModFiles []string

	_ = filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip directories we can't read
		}

		// Skip hidden directories and vendor
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// Collect go.mod files (excluding the root one)
		if d.Name() == "go.mod" && path != filepath.Join(rootPath, "go.mod") {
			goModFiles = append(goModFiles, path)
		}

		return nil
	})

	return goModFiles
}

func isGitRepo(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func hasUncommittedChanges(repoPath string) bool {
	// Use git status --porcelain to check for changes
	cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

func analyzeGoMod(goModPath string) (moduleName string, replaceCount int, dependencies []string) {
	file, err := os.Open(goModPath)
	if err != nil {
		return "", 0, nil
	}
	defer func() {
		_ = file.Close()
	}()

	s := bufio.NewScanner(file)
	inReplaceBlock := false
	inRequireBlock := false

	for s.Scan() {
		line := strings.TrimSpace(s.Text())

		// Get module name
		if mod, found := strings.CutPrefix(line, "module "); found {
			moduleName = strings.TrimSpace(mod)
		}

		// Count replace directives
		if strings.HasPrefix(line, "replace ") && !strings.HasPrefix(line, "replace (") {
			replaceCount++
		}

		// Handle replace block
		if strings.HasPrefix(line, "replace (") {
			inReplaceBlock = true
			continue
		}
		if inReplaceBlock {
			if line == ")" {
				inReplaceBlock = false
				continue
			}
			if line != "" && !strings.HasPrefix(line, "//") {
				replaceCount++
			}
		}

		// Parse single-line require
		if strings.HasPrefix(line, "require ") && !strings.HasPrefix(line, "require (") {
			if dep := parseRequireLine(strings.TrimPrefix(line, "require ")); dep != "" {
				dependencies = append(dependencies, dep)
			}
		}

		// Handle require block
		if strings.HasPrefix(line, "require (") {
			inRequireBlock = true
			continue
		}
		if inRequireBlock {
			if line == ")" {
				inRequireBlock = false
				continue
			}
			if dep := parseRequireLine(line); dep != "" {
				dependencies = append(dependencies, dep)
			}
		}
	}

	return moduleName, replaceCount, dependencies
}

// parseRequireLine extracts the module path from a require line.
// Input: "github.com/foo/bar v1.2.3" or "github.com/foo/bar v1.2.3 // indirect"
// Output: "github.com/foo/bar"
func parseRequireLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "//") {
		return ""
	}
	// Split on whitespace, first part is module path
	parts := strings.Fields(line)
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

// moduleMatchesPath checks if the module name ends with the directory name.
// For example: github.com/grokify/gitscan should match directory "gitscan"
func moduleMatchesPath(moduleName, dirName string) bool {
	// Get the last segment of the module path
	parts := strings.Split(moduleName, "/")
	if len(parts) == 0 {
		return false
	}
	lastPart := parts[len(parts)-1]
	return lastPart == dirName
}
