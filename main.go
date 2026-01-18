package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grokify/gitscan/scanner"
)

const progressBarWidth = 40

func main() {
	var (
		dirPath     string
		showClean   bool
		showSummary bool
		format      string
		depFilter   string
		recurse     bool
	)

	flag.StringVar(&dirPath, "dir", "", "Directory to scan (required)")
	flag.BoolVar(&showClean, "show-clean", false, "Show repos with no issues")
	flag.BoolVar(&showSummary, "summary", true, "Show summary at the end")
	flag.StringVar(&format, "format", "list", "Output format: list or table")
	flag.StringVar(&depFilter, "dep", "", "Filter repos by dependency (module path)")
	flag.BoolVar(&recurse, "recurse", false, "Recursively search for nested go.mod files")
	flag.Parse()

	// If no -dir flag, check for positional argument
	if dirPath == "" && flag.NArg() > 0 {
		dirPath = flag.Arg(0)
	}

	if dirPath == "" {
		fmt.Fprintln(os.Stderr, "Error: directory path required")
		fmt.Fprintln(os.Stderr, "Usage: gitscan [-dir] <directory> [-show-clean] [-summary] [-format list|table]")
		os.Exit(1)
	}

	// Validate format
	if format != "list" && format != "table" {
		fmt.Fprintf(os.Stderr, "Error: invalid format %q, must be 'list' or 'table'\n", format)
		os.Exit(1)
	}

	// Expand ~ to home directory
	if dirPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}
		dirPath = filepath.Join(home, dirPath[1:])
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(1)
	}

	// Check if directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error accessing directory: %v\n", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a directory\n", absPath)
		os.Exit(1)
	}

	fmt.Printf("Scanning: %s\n", absPath)

	// Count directories first
	total, err := scanner.CountDirectories(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error counting directories: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found %d directories to scan\n\n", total)

	// Progress callback
	progressFn := func(current, total int, name string) {
		percent := float64(current) / float64(total) * 100
		filled := int(float64(progressBarWidth) * float64(current) / float64(total))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", progressBarWidth-filled)

		// Truncate name if too long
		displayName := name
		if len(displayName) > 30 {
			displayName = displayName[:27] + "..."
		}

		// \r returns to start of line, overwriting previous output
		fmt.Printf("\r[%s] %3.0f%% (%d/%d) %-30s", bar, percent, current, total, displayName)
	}

	opts := scanner.ScanOptions{
		Recurse: recurse,
	}
	results, err := scanner.ScanDirectoryWithProgress(absPath, progressFn, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError scanning directory: %v\n", err)
		os.Exit(1)
	}

	// Clear the progress line and move to next line
	fmt.Printf("\r%s\r", strings.Repeat(" ", 100))
	fmt.Println("Scan complete!")

	// Display results based on format
	var (
		totalRepos       int
		reposWithIssues  int
		uncommittedCount int
		replaceCount     int
		mismatchCount    int
		depMatchCount    int
	)

	if format == "table" {
		printTableHeader(depFilter != "", recurse)
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

		// Determine if we should show this result
		shouldShow := false
		if depFilter != "" {
			// When filtering by dependency, only show matches
			shouldShow = hasDep
		} else {
			// Normal mode: show issues or clean if requested
			shouldShow = hasIssues || showClean
		}

		if shouldShow {
			rowNum++
			if format == "table" {
				printTableRow(rowNum, result, depFilter != "", recurse)
			} else {
				printResult(result, depFilter, recurse)
			}
		}
	}

	fmt.Println()
	if showSummary {
		fmt.Println("----------------------------------------")
		if depFilter != "" {
			fmt.Printf("Summary: %d repos scanned, %d depend on %s\n", totalRepos, depMatchCount, depFilter)
		} else {
			fmt.Printf("Summary: %d repos scanned, %d with issues\n", totalRepos, reposWithIssues)
			fmt.Printf("  - Uncommitted changes: %d\n", uncommittedCount)
			fmt.Printf("  - Replace directives:  %d\n", replaceCount)
			fmt.Printf("  - Module mismatches:   %d\n", mismatchCount)
		}
	}
}

func printTableHeader(showDep, showNested bool) {
	fmt.Println()
	if showDep {
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

func printTableRow(num int, r scanner.RepoResult, showDep, showNested bool) {
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

func printResult(r scanner.RepoResult, depFilter string, showNested bool) {
	if depFilter != "" {
		// Dependency-focused output
		fmt.Printf("[DEP] %s\n", r.Name)
		if r.HasGoMod {
			fmt.Printf("    - Module: %s\n", r.ModuleName)
		}
		if showNested && len(r.GoModFiles) > 0 {
			fmt.Println("    - Nested modules:")
			for _, gm := range r.GoModFiles {
				fmt.Printf("      - %s (%s)\n", gm.ModuleName, gm.Path)
			}
		}
		fmt.Println()
		return
	}

	// Standard output
	hasIssues := r.HasUncommittedChanges || r.HasReplaceDirectives || r.HasModuleMismatch

	if hasIssues {
		fmt.Printf("[!] %s\n", r.Name)
	} else {
		fmt.Printf("[OK] %s\n", r.Name)
	}

	if !r.IsGitRepo {
		fmt.Println("    - Not a git repository")
	}
	if !r.HasGoMod {
		fmt.Println("    - No go.mod file")
	}

	if r.HasUncommittedChanges {
		fmt.Println("    - Has uncommitted changes")
	}
	if r.HasReplaceDirectives {
		fmt.Printf("    - Has replace directives (%d)\n", r.ReplaceCount)
	}
	if r.HasModuleMismatch {
		fmt.Printf("    - Module name mismatch: %s\n", r.ModuleName)
	}

	fmt.Println()
}
