# gitscan v0.1.0 Release Notes

**Release Date:** 2026-01-17

## Overview

gitscan is a CLI tool to scan multiple Git repositories and identify repos that need attention. It helps prioritize which repos to update, commit, and push - particularly useful for developers managing many Go projects.

## Highlights

- CLI tool to scan multiple Git repositories and identify repos needing attention
- Zero external dependencies - uses only Go standard library

## Installation

```bash
go install github.com/grokify/gitscan@v0.1.0
```

Or build from source:

```bash
git clone https://github.com/grokify/gitscan.git
cd gitscan
go build -o gitscan .
```

## Features

### Repository Analysis

- **Uncommitted Changes** - Detects modified, added, or deleted files using `git status --porcelain`
- **Replace Directives** - Parses `go.mod` for `replace` directives (both single-line and block format), which may indicate local development dependencies
- **Module Name Mismatch** - Compares the module name in `go.mod` with the directory name to identify renamed or copied repos

### Dependency Search

- Filter repos by dependency with `-dep` flag to find all repos depending on a specific module
- Recursive nested go.mod search with `-recurse` flag for monorepo support

### User Experience

- Progress bar with percentage, count, and current repo name during scanning
- Two output formats: human-readable list (default) and compact markdown table
- Summary statistics showing counts of uncommitted changes, replace directives, and mismatches

## Usage

```bash
# Scan all repos in a directory
gitscan ~/go/src/github.com/grokify

# Output as markdown table
gitscan -format table ~/go/src/github.com/grokify

# Find repos depending on a module
gitscan -dep github.com/grokify/mogo ~/go/src/github.com/grokify

# Include nested go.mod files
gitscan -dep github.com/grokify/mogo -recurse ~/go/src/github.com/grokify
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-dir` | (required) | Directory containing repos to scan |
| `-format` | `list` | Output format: `list` or `table` |
| `-show-clean` | `false` | Show repos with no issues |
| `-summary` | `true` | Show summary at the end |
| `-dep` | (none) | Filter repos by dependency (module path) |
| `-recurse` | `false` | Recursively search for nested go.mod files |

## Use Cases

- **Pre-push audit** - Identify repos with uncommitted work before leaving for vacation
- **Dependency cleanup** - Find repos with local `replace` directives that need resolution
- **Repo hygiene** - Detect copied/renamed repos with mismatched module names
- **Breaking changes** - Find all repos to update before releasing library changes
- **Security patches** - Locate repos using vulnerable dependencies

## License

MIT
