# gitscan v0.4.0 Release Notes

**Release Date:** 2026-02-21

## Overview

gitscan v0.4.0 restructures the CLI into focused Cobra subcommands, fixing the issue where `--since` and `--dep` flags were mutually exclusive but not documented as such. The root command now focuses solely on issue detection, while filtering is handled by dedicated `since` and `dep` subcommands. This release also adds Homebrew distribution.

## Highlights

- Restructured CLI into focused Cobra subcommands
- New `since` subcommand with optional `--dep` flag for AND logic
- New `dep` subcommand for dependency filtering
- Homebrew distribution via `brew install gitscan`

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap grokify/tap
brew install gitscan
```

### Go Install

```bash
go install github.com/grokify/gitscan@v0.4.0
```

### Build from Source

```bash
git clone https://github.com/grokify/gitscan.git
cd gitscan
git checkout v0.4.0
go build -o gitscan .
```

## Breaking Changes

The `--since` and `--dep` flags have been removed from the root command and are now subcommands:

| Before (v0.3.0) | After (v0.4.0) |
|-----------------|----------------|
| `gitscan --since 7d ~/repos` | `gitscan since 7d ~/repos` |
| `gitscan --dep github.com/foo/bar ~/repos` | `gitscan dep github.com/foo/bar ~/repos` |

## New Command Structure

```bash
gitscan [dir]                    # Scan for issues (uncommitted, replace, mismatch)
gitscan since <duration> [dir]   # Filter by modification time
gitscan dep <module> [dir]       # Filter by dependency
gitscan order [dir]              # Show repos in dependency order
```

## New Features

### `since` Subcommand

Filter repos by modification time:

```bash
# Repos modified in last 7 days
gitscan since 7d ~/go/src/github.com/grokify

# Combined with dependency filter (AND logic)
gitscan since 7d --dep github.com/grokify/mogo ~/go/src/github.com/grokify
```

The `--dep` flag on the `since` command applies AND logic - showing only repos that are both recently modified AND depend on the specified module.

### `dep` Subcommand

Filter repos by dependency:

```bash
# Find all repos depending on a module
gitscan dep github.com/grokify/mogo ~/go/src/github.com/grokify

# Include nested go.mod files
gitscan dep github.com/grokify/mogo -r ~/go/src/github.com/grokify
```

### Homebrew Distribution

gitscan is now available via Homebrew:

```bash
brew tap grokify/tap
brew install gitscan
```

Cross-platform binaries are built automatically via GoReleaser for:

- macOS (amd64, arm64)
- Linux (amd64, arm64)
- Windows (amd64, arm64)

## Code Changes

- New `cmd/shared.go` with common helpers: `resolvePath()`, `createGitBackend()`, `parseDuration()`
- New `cmd/since.go` for the `since` subcommand
- New `cmd/dep.go` for the `dep` subcommand
- Simplified `cmd/root.go` to focus on issue scanning only
- Added `.goreleaser.yaml` for release automation
- Added `.github/workflows/release.yaml` for GitHub Actions

## Migration Guide

1. Replace `gitscan --since <duration>` with `gitscan since <duration>`
2. Replace `gitscan --dep <module>` with `gitscan dep <module>`
3. To combine both filters, use: `gitscan since <duration> --dep <module>`

## License

MIT
