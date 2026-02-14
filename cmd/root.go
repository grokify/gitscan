package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/grokify/gitscan/scanner"
	"github.com/grokify/mogo/fmt/progress"
	"github.com/spf13/cobra"
)

const (
	version          = "0.2.0"
	progressBarWidth = 40
)

var (
	dirPath     string
	showClean   bool
	showSummary bool
	format      string
	depFilter   string
	recurse     bool
	sinceStr    string
)

var rootCmd = &cobra.Command{
	Use:   "gitscan [directory]",
	Short: "Scan git repositories for common issues",
	Long: `gitscan scans multiple Git repositories and identifies repos that need attention.
It helps developers prioritize which repositories to update, commit, and push
by detecting uncommitted changes, replace directives, and module mismatches.`,
	Version: version,
	Args:    cobra.MaximumNArgs(1),
	RunE:    runScan,
}

func init() {
	rootCmd.Flags().StringVarP(&dirPath, "dir", "d", "", "Directory to scan")
	rootCmd.Flags().BoolVar(&showClean, "show-clean", false, "Show repos with no issues")
	rootCmd.Flags().BoolVar(&showSummary, "summary", true, "Show summary at the end")
	rootCmd.Flags().StringVarP(&format, "format", "f", "list", "Output format: list or table")
	rootCmd.Flags().StringVar(&depFilter, "dep", "", "Filter repos by dependency (module path)")
	rootCmd.Flags().BoolVarP(&recurse, "recurse", "r", false, "Recursively search for nested go.mod files")
	rootCmd.Flags().StringVarP(&sinceStr, "since", "s", "", "Filter repos modified within duration (e.g., 7d, 14d, 2w, 1m)")
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runScan(cmd *cobra.Command, args []string) error {
	// Handle positional argument
	if len(args) > 0 && dirPath == "" {
		dirPath = args[0]
	}

	if dirPath == "" {
		return fmt.Errorf("directory path required\nUsage: gitscan [directory] or gitscan -d <directory>")
	}

	// Validate format
	if format != "list" && format != "table" {
		return fmt.Errorf("invalid format %q, must be 'list' or 'table'", format)
	}

	// Parse since duration
	var sinceDuration time.Duration
	if sinceStr != "" {
		var err error
		sinceDuration, err = parseDuration(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %v\nValid formats: 7d (days), 2w (weeks), 1m (months), 24h (hours)", sinceStr, err)
		}
	}

	// Expand ~ to home directory
	if len(dirPath) > 0 && dirPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("error getting home directory: %w", err)
		}
		dirPath = filepath.Join(home, dirPath[1:])
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return fmt.Errorf("error resolving path: %w", err)
	}

	// Check if directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("error accessing directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", absPath)
	}

	fmt.Printf("Scanning: %s\n", absPath)

	// Count directories first
	total, err := scanner.CountDirectories(absPath)
	if err != nil {
		return fmt.Errorf("error counting directories: %w", err)
	}
	fmt.Printf("Found %d directories to scan\n\n", total)

	// Progress renderer
	renderer := progress.NewSingleStageRenderer(os.Stdout).WithBarWidth(progressBarWidth)

	// Progress callback
	progressFn := func(current, total int, name string) {
		renderer.Update(current, total, name)
	}

	opts := scanner.ScanOptions{
		Recurse:      recurse,
		CheckModTime: sinceDuration > 0, // Only compute mod time if filtering by it
	}
	results, err := scanner.ScanDirectoryWithProgress(absPath, progressFn, opts)
	if err != nil {
		return fmt.Errorf("error scanning directory: %w", err)
	}

	// Clear the progress line and show completion
	renderer.Done("Scan complete!")

	// Sort results alphabetically by name
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	// Calculate max name length for alignment
	maxNameLen := 0
	for _, r := range results {
		if len(r.Name) > maxNameLen {
			maxNameLen = len(r.Name)
		}
	}

	// Display results based on format
	var (
		totalRepos       int
		reposWithIssues  int
		uncommittedCount int
		replaceCount     int
		mismatchCount    int
		depMatchCount    int
		sinceMatchCount  int
	)

	if format == "table" {
		printTableHeader(depFilter != "", recurse, sinceDuration > 0)
	}

	rowNum := 0
	for _, result := range results {
		totalRepos++
		hasIssues := result.HasUncommittedChanges || result.HasReplaceDirectives || result.HasModuleMismatch

		if hasIssues {
			reposWithIssues++
			if result.HasUncommittedChanges {
				uncommittedCount++
			}
			if result.HasReplaceDirectives {
				replaceCount++
			}
			if result.HasModuleMismatch {
				mismatchCount++
			}
		}

		// Check dependency filter
		hasDep := depFilter != "" && result.HasDependency(depFilter)
		if hasDep {
			depMatchCount++
		}

		// Check since filter
		matchesSince := sinceDuration == 0 || result.ModifiedSince(sinceDuration)
		if sinceDuration > 0 && matchesSince {
			sinceMatchCount++
		}

		// Determine if we should show this result
		shouldShow := false
		if sinceDuration > 0 {
			// When filtering by time, only show repos modified within the duration
			shouldShow = matchesSince
		} else if depFilter != "" {
			// When filtering by dependency, only show matches
			shouldShow = hasDep
		} else {
			// Normal mode: show issues or clean if requested
			shouldShow = hasIssues || showClean
		}

		if shouldShow {
			rowNum++
			if format == "table" {
				printTableRow(rowNum, result, depFilter != "", recurse, sinceDuration > 0)
			} else {
				internalDeps := scanner.GetInternalDeps(result, results)
				printResult(rowNum, result, depFilter, recurse, sinceDuration > 0, maxNameLen, internalDeps)
			}
		}
	}

	fmt.Println()
	if showSummary {
		fmt.Println("----------------------------------------")
		if sinceDuration > 0 {
			fmt.Printf("Summary: %d repos scanned, %d modified within %s\n", totalRepos, sinceMatchCount, sinceStr)
		} else if depFilter != "" {
			fmt.Printf("Summary: %d repos scanned, %d depend on %s\n", totalRepos, depMatchCount, depFilter)
		} else {
			fmt.Printf("Summary: %d repos scanned, %d with issues\n", totalRepos, reposWithIssues)
			fmt.Printf("  - Uncommitted changes: %d\n", uncommittedCount)
			fmt.Printf("  - Replace directives:  %d\n", replaceCount)
			fmt.Printf("  - Module mismatches:   %d\n", mismatchCount)
		}
	}

	return nil
}

func printTableHeader(showDep, showNested, showSince bool) {
	fmt.Println()
	if showSince {
		fmt.Println("| # | Repository | Last Modified |")
		fmt.Println("|---|------------|---------------|")
	} else if showDep {
		if showNested {
			fmt.Println("| # | Repository | Module | Location |")
			fmt.Println("|---|------------|--------|----------|")
		} else {
			fmt.Println("| # | Repository | Module |")
			fmt.Println("|---|------------|--------|")
		}
	} else {
		fmt.Println("| # | Repository | Uncommitted | Replace | Mismatch | Git | go.mod |")
		fmt.Println("|---|------------|-------------|---------|----------|-----|--------|")
	}
}

func printTableRow(num int, r scanner.RepoResult, showDep, showNested, showSince bool) {
	if showSince {
		// Show time-focused output
		modTime := r.LatestModTime.Format("2006-01-02 15:04")
		fmt.Printf("| %d | %s | %s |\n", num, r.Name, modTime)
		return
	}

	if showDep {
		// Show dependency-focused output
		if showNested && len(r.GoModFiles) > 0 {
			// Show root module
			if r.HasGoMod {
				fmt.Printf("| %d | %s | %s | (root) |\n", num, r.Name, r.ModuleName)
			}
			// Show nested modules
			for _, gm := range r.GoModFiles {
				fmt.Printf("| | | %s | %s |\n", gm.ModuleName, gm.Path)
			}
		} else {
			fmt.Printf("| %d | %s | %s |\n", num, r.Name, r.ModuleName)
		}
		return
	}

	// Standard output
	uncommitted := ""
	if r.HasUncommittedChanges {
		uncommitted = "X"
	}

	replace := ""
	if r.HasReplaceDirectives {
		replace = fmt.Sprintf("%d", r.ReplaceCount)
	}

	mismatch := ""
	if r.HasModuleMismatch {
		mismatch = "X"
	}

	git := "Y"
	if !r.IsGitRepo {
		git = "-"
	}

	gomod := "Y"
	if !r.HasGoMod {
		gomod = "-"
	}

	fmt.Printf("| %d | %s | %s | %s | %s | %s | %s |\n",
		num, r.Name, uncommitted, replace, mismatch, git, gomod)
}

func printResult(num int, r scanner.RepoResult, depFilter string, showNested, showSince bool, maxNameLen int, internalDeps []string) {
	if showSince {
		// Time-focused output: aligned date with internal dependencies
		modTime := r.LatestModTime.Format("2006-01-02 15:04")
		depStr := ""
		if len(internalDeps) > 0 {
			depStr = fmt.Sprintf(" (depends on: %s)", strings.Join(internalDeps, ", "))
		}
		fmt.Printf("%3d. %-*s  %s%s\n", num, maxNameLen, r.Name, modTime, depStr)
		return
	}

	if depFilter != "" {
		// Dependency-focused output: single line
		if showNested && len(r.GoModFiles) > 0 {
			fmt.Printf("%3d. %-*s  [%s + %d nested]\n", num, maxNameLen, r.Name, r.ModuleName, len(r.GoModFiles))
		} else {
			fmt.Printf("%3d. %-*s  [%s]\n", num, maxNameLen, r.Name, r.ModuleName)
		}
		return
	}

	// Standard output: single line with issue indicators
	var issues []string
	if r.HasUncommittedChanges {
		issues = append(issues, "uncommitted")
	}
	if r.HasReplaceDirectives {
		issues = append(issues, fmt.Sprintf("replace:%d", r.ReplaceCount))
	}
	if r.HasModuleMismatch {
		issues = append(issues, "mismatch")
	}
	if !r.IsGitRepo {
		issues = append(issues, "no-git")
	}
	if !r.HasGoMod {
		issues = append(issues, "no-gomod")
	}

	depStr := ""
	if len(internalDeps) > 0 {
		depStr = fmt.Sprintf(" (depends on: %s)", strings.Join(internalDeps, ", "))
	}

	if len(issues) > 0 {
		fmt.Printf("%3d. %-*s  [%s]%s\n", num, maxNameLen, r.Name, joinIssues(issues), depStr)
	} else {
		fmt.Printf("%3d. %-*s%s\n", num, maxNameLen, r.Name, depStr)
	}
}

func joinIssues(issues []string) string {
	result := ""
	for i, issue := range issues {
		if i > 0 {
			result += ", "
		}
		result += issue
	}
	return result
}

// parseDuration parses duration strings like "7d", "2w", "1m", "24h".
// Supported units: h (hours), d (days), w (weeks), m (months, 30 days).
func parseDuration(s string) (time.Duration, error) {
	// Try standard Go duration first (e.g., "24h", "1h30m")
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Parse custom formats: 7d, 2w, 1m
	re := regexp.MustCompile(`^(\d+)([dwm])$`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("invalid duration format")
	}

	value, _ := strconv.Atoi(matches[1])
	unit := matches[2]

	switch unit {
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	case "w":
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	case "m":
		return time.Duration(value) * 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown unit: %s", unit)
	}
}
