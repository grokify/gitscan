# gitscan v0.2.0 Release Notes

**Release Date:** 2026-02-14

## Overview

gitscan v0.2.0 introduces the `order` subcommand for dependency-aware release ordering, time-based filtering, and significant performance improvements through parallel scanning.

## Highlights

- New `order` subcommand for topological dependency ordering
- Time-based filtering with `-since` flag
- Parallel scanning for significantly faster performance

## Installation

```bash
go install github.com/grokify/gitscan@v0.2.0
```

Or build from source:

```bash
git clone https://github.com/grokify/gitscan.git
cd gitscan
git checkout v0.2.0
go build -o gitscan .
```

## New Features

### Order Subcommand

The new `order` subcommand displays repositories in topological dependency order, showing which repos should be updated first based on their internal dependencies.

```bash
# Show repos in dependency order
gitscan order ~/go/src/github.com/grokify

# Filter by modification time
gitscan order -s 7d ~/go/src/github.com/grokify

# Include transitive dependents
gitscan order -s 7d -t ~/go/src/github.com/grokify

# Only show repos needing push
gitscan order -s 7d -t -u ~/go/src/github.com/grokify
```

Output shows dependencies first, then dependents:

```
Update order (dependencies first):
----------------------------------
  1. mogo                  2026-02-08 12:28
  2. gogithub              2026-02-07 08:09 (depends on: mogo)
  3. goauth                2026-02-09 19:38 (depends on: mogo)
  4. gogoogle              2026-02-09 17:31 (depends on: goauth, mogo)

Total: 4 repos in dependency order
```

### Time-Based Filtering

Filter repos by modification time using human-friendly duration formats:

```bash
gitscan -s 7d ~/go/src/github.com/grokify   # Last 7 days
gitscan -s 2w ~/go/src/github.com/grokify   # Last 2 weeks
gitscan -s 1m ~/go/src/github.com/grokify   # Last month
```

### Transitive Dependency Tracking

With `-t` flag, include repos that transitively depend on modified repos, even if they weren't directly modified:

```bash
gitscan order -s 7d -t ~/go/src/github.com/grokify
```

### Unpushed Commit Detection

With `-u` flag, filter to only show repos with uncommitted changes or unpushed commits:

```bash
gitscan order -s 7d -u ~/go/src/github.com/grokify
```

### Parallel Scanning

Scanning now uses a goroutine worker pool (defaults to GOMAXPROCS) for significantly faster scanning of large directory trees. Expensive operations are performed lazily only when needed.

## Changes

### CLI Migration to Cobra

The CLI has been migrated from the standard `flag` package to [Cobra](https://github.com/spf13/cobra) for better UX:

- Short flags: `-d` (dir), `-f` (format), `-r` (recurse), `-s` (since)
- Built-in `--help` and `--version` flags
- Subcommand support (`gitscan order`)

### Output Improvements

- Numbered list output with alphabetical sorting
- Aligned columns for better readability
- Internal dependency display: `(depends on: x, y)`

## New Flags

### Root Command

| Flag | Short | Description |
|------|-------|-------------|
| `--since` | `-s` | Filter repos modified within duration (e.g., `7d`, `2w`, `1m`) |

### Order Subcommand

| Flag | Short | Description |
|------|-------|-------------|
| `--transitive` | `-t` | Include repos that transitively depend on modified repos |
| `--unpushed` | `-u` | Only show repos with uncommitted changes or unpushed commits |

## Dependencies

- Added `github.com/spf13/cobra` for CLI framework
- Added `github.com/grokify/mogo/fmt/progress` for progress bar rendering

## Use Cases

- **Release ordering**: Determine correct order to update interdependent modules
- **CI/CD planning**: Identify which repos need rebuilding after dependency updates
- **Pre-push audit**: Find repos with unpushed work using `-u` flag
- **Recent activity**: Filter to recently modified repos with `-s` flag

## License

MIT
