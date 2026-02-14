# gitscan

[![Build Status][build-status-svg]][build-status-url]
[![Lint Status][lint-status-svg]][lint-status-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Visualization][viz-svg]][viz-url]

A CLI tool to scan multiple Git repositories and identify repos that need attention. Helps prioritize which repos to update, commit, and push.

## Installation

```bash
go install github.com/grokify/gitscan@latest
```

Or build from source:

```bash
git clone https://github.com/grokify/gitscan.git
cd gitscan
go build -o gitscan .
```

## Usage

```bash
gitscan <directory>
gitscan -d <directory>
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--dir` | `-d` | (required) | Directory containing repos to scan |
| `--format` | `-f` | `list` | Output format: `list` or `table` |
| `--since` | `-s` | (none) | Filter repos modified within duration (e.g., `7d`, `2w`, `1m`) |
| `--recurse` | `-r` | `false` | Recursively search for nested go.mod files |
| `--dep` | | (none) | Filter repos by dependency (module path) |
| `--show-clean` | | `false` | Show repos with no issues |
| `--summary` | | `true` | Show summary at the end |
| `--help` | `-h` | | Show help |
| `--version` | `-v` | | Show version |

### Examples

```bash
# Scan all repos in a directory
gitscan ~/go/src/github.com/grokify

# Filter repos modified in last 7 days
gitscan -s 7d ~/go/src/github.com/grokify

# Output as markdown table (compact view)
gitscan -f table ~/go/src/github.com/grokify

# Show all repos including clean ones
gitscan --show-clean ~/projects

# Find repos depending on a module
gitscan --dep github.com/grokify/mogo ~/go/src/github.com/grokify
```

## Order Subcommand

The `order` subcommand shows repos in topological dependency order - dependencies first, then dependents. This helps determine the correct order to update and release Go modules.

```bash
gitscan order <directory>
```

### Order Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--dir` | `-d` | (required) | Directory containing repos to scan |
| `--since` | `-s` | (none) | Filter repos modified within duration |
| `--transitive` | `-t` | `false` | Include repos that transitively depend on modified repos |
| `--unpushed` | `-u` | `false` | Only show repos with uncommitted changes or unpushed commits |

### Order Examples

```bash
# Show all repos in dependency order
gitscan order ~/go/src/github.com/grokify

# Repos modified in last 7 days, in dependency order
gitscan order -s 7d ~/go/src/github.com/grokify

# Include transitive dependents (repos depending on modified repos)
gitscan order -s 7d -t ~/go/src/github.com/grokify

# Only show repos that need to be pushed
gitscan order -s 7d -t -u ~/go/src/github.com/grokify
```

### Order Output

```
Update order (dependencies first):
----------------------------------
  1. mogo                  2026-02-08 12:28
  2. gogithub              2026-02-07 08:09 (depends on: mogo)
  3. goauth                2026-02-09 19:38 (depends on: mogo)
  4. gogoogle              2026-02-09 17:31 (depends on: goauth, mogo)
  5. go-aha                2026-02-09 02:15 (depends on: goauth, gogoogle, mogo)

Total: 5 repos in dependency order
```

## Checks Performed

For each direct subdirectory, gitscan checks:

1. **Uncommitted Changes** - Detects modified, added, or deleted files using `git status --porcelain`

2. **Replace Directives** - Parses `go.mod` for `replace` directives (both single-line and block format), which may indicate local development dependencies that shouldn't be committed

3. **Module Name Mismatch** - Compares the module name in `go.mod` with the directory name to identify renamed or copied repos

4. **Unpushed Commits** - Detects commits that haven't been pushed to remote (with `-u` flag)

## Output Format

During scanning, a progress bar shows real-time status:

```
Scanning: /Users/you/go/src/github.com/grokify
Found 584 directories to scan

[████████████████░░░░░░░░░░░░░░░░░░░░░░░░]  42% (245/584) my-current-repo
```

### List Format (default)

Repos are shown in a numbered list with issues and internal dependencies:

```
  1. mogo                  2026-02-08 12:28
  2. gogithub              2026-02-07 08:09 (depends on: mogo)
  3. my-service            2026-02-10 15:30 [uncommitted, replace:2]

Summary: 100 repos scanned, 25 modified within 7d
```

### Table Format (`-f table`)

Compact markdown table with one repo per row:

```
| # | Repository | Uncommitted | Replace | Mismatch | Git | go.mod |
|---|------------|-------------|---------|----------|-----|--------|
| 1 | omnistorage |  |  | X | Y | Y |
| 2 | omnistorage-github | X |  |  | Y | - |
| 3 | structured-changelog | X |  |  | Y | Y |
| 5 | structured-roadmap |  | 5 |  | - | Y |
```

Column legend:

- **Uncommitted**: `X` = has uncommitted changes
- **Replace**: number of replace directives in go.mod
- **Mismatch**: `X` = module name doesn't match directory
- **Git**: `Y` = is a git repo, `-` = not a git repo
- **go.mod**: `Y` = has go.mod, `-` = no go.mod

## Finding Dependents

When making breaking changes to a library, find all local repos that depend on it:

```bash
# Find repos depending on a module
gitscan --dep github.com/grokify/gogithub ~/go/src/github.com/grokify

# Include nested go.mod files (monorepos, nested modules)
gitscan --dep github.com/grokify/gogithub -r ~/go/src/github.com/grokify

# Output as table
gitscan --dep github.com/grokify/mogo -f table ~/go/src/github.com/grokify
```

## Performance

gitscan uses parallel scanning with a goroutine worker pool (defaults to GOMAXPROCS workers) for fast scanning of large directory trees. Expensive operations like modification time calculation and unpushed commit detection are performed lazily only when needed.

## Use Cases

- **Pre-push audit**: Identify repos with uncommitted work before leaving for vacation
- **Dependency cleanup**: Find repos with local `replace` directives that need resolution
- **Repo hygiene**: Detect copied/renamed repos with mismatched module names
- **Breaking changes**: Find all repos to update before releasing library changes
- **Security patches**: Locate repos using vulnerable dependencies
- **Release ordering**: Determine correct order to update and release interdependent modules
- **Prioritization**: Focus on repos that need immediate attention

## License

MIT

 [build-status-svg]: https://github.com/grokify/gitscan/actions/workflows/ci.yaml/badge.svg?branch=main
 [build-status-url]: https://github.com/grokify/gitscan/actions/workflows/ci.yaml
 [lint-status-svg]: https://github.com/grokify/gitscan/actions/workflows/lint.yaml/badge.svg?branch=main
 [lint-status-url]: https://github.com/grokify/gitscan/actions/workflows/lint.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/grokify/gitscan
 [goreport-url]: https://goreportcard.com/report/github.com/grokify/gitscan
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/grokify/gitscan
 [docs-godoc-url]: https://pkg.go.dev/github.com/grokify/gitscan
 [viz-svg]: https://img.shields.io/badge/visualizaton-Go-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=grokify%2Fgitscan
 [loc-svg]: https://tokei.rs/b1/github/grokify/gitscan
 [repo-url]: https://github.com/grokify/gitscan
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/grokify/gitscan/blob/master/LICENSE
