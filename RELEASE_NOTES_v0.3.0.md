# gitscan v0.3.0 Release Notes

**Release Date:** 2026-02-14

## Overview

gitscan v0.3.0 introduces a pluggable git backend architecture with both go-git (pure Go) and git CLI implementations. The git CLI remains the default for optimal performance, while go-git is available as an option for environments without the git binary.

## Highlights

- Git backend abstraction with go-git and CLI implementations
- Optional `--go-git` flag for pure Go git operations (no process spawning)

## Installation

```bash
go install github.com/grokify/gitscan@v0.3.0
```

Or build from source:

```bash
git clone https://github.com/grokify/gitscan.git
cd gitscan
git checkout v0.3.0
go build -o gitscan .
```

## New Features

### Git Backend Abstraction

The scanner package now uses a `GitBackend` interface for pluggable git implementations:

- `GitBackend` - Interface with `IsRepo()` and `GetStatus()` methods
- `GoGitBackend` - Pure Go implementation using go-git library
- `CLIGitBackend` - Git CLI implementation (default)

### go-git Backend Option

Use the `--go-git` flag to enable pure Go git operations:

```bash
# Use go-git library (pure Go, no process spawning)
gitscan --go-git ~/go/src/github.com/grokify

# Works with order subcommand too
gitscan order --go-git ~/go/src/github.com/grokify
```

This is useful in environments where the git binary is unavailable or when you want to avoid external process spawning.

## Performance Comparison

| Backend | Speed (600 repos) | Use Case |
|---------|-------------------|----------|
| git CLI (default) | ~2.5s | Most use cases, full compatibility |
| go-git (`--go-git`) | ~10s | Environments without git binary |

The git CLI backend is recommended for most use cases due to its significantly faster performance.

## Changes

### Default Backend

- Git CLI is now the default backend (faster and more compatible)
- go-git is available as an opt-in alternative via `--go-git` flag

### Scanner Package

- New `GitBackend` interface for pluggable git implementations
- `ScanOptions` now includes optional `GitBackend` field

## New Flags

| Flag | Description |
|------|-------------|
| `--go-git` | Use go-git library instead of git CLI |

Available on both `scan` and `order` commands.

## Dependencies

- Added `github.com/go-git/go-git/v5` for pure Go git operations

## Use Cases

- **Containerized environments**: Use `--go-git` when git binary is not installed
- **Embedded applications**: Pure Go implementation with no external dependencies
- **Cross-platform builds**: Avoid git CLI compatibility issues

## License

MIT
