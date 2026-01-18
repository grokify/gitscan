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
gitscan -dir <directory>
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-dir` | (required) | Directory containing repos to scan |
| `-format` | `list` | Output format: `list` or `table` |
| `-show-clean` | `false` | Show repos with no issues |
| `-summary` | `true` | Show summary at the end |
| `-dep` | (none) | Filter repos by dependency (module path) |
| `-recurse` | `false` | Recursively search for nested go.mod files |

### Examples

```bash
# Scan all repos in a directory
gitscan ~/go/src/github.com/grokify

# Output as markdown table (compact view)
gitscan -format table ~/go/src/github.com/grokify

# Show all repos including clean ones
gitscan -show-clean ~/projects

# Scan without summary
gitscan -summary=false ~/repos
```

## Checks Performed

For each direct subdirectory, gitscan checks:

1. **Uncommitted Changes** - Detects modified, added, or deleted files using `git status --porcelain`

2. **Replace Directives** - Parses `go.mod` for `replace` directives (both single-line and block format), which may indicate local development dependencies that shouldn't be committed

3. **Module Name Mismatch** - Compares the module name in `go.mod` with the directory name to identify renamed or copied repos

## Output Format

During scanning, a progress bar shows real-time status:

```
Scanning: /Users/you/go/src/github.com/grokify
Found 584 directories to scan

[████████████████░░░░░░░░░░░░░░░░░░░░░░░░]  42% (245/584) my-current-repo
```

### List Format (default)

Repos with issues are marked with `[!]`, clean repos with `[OK]`:

```
Scan complete!

[!] my-repo
    - Has uncommitted changes
    - Has replace directives (2)
    - Module name mismatch: github.com/other/name

[OK] clean-repo

----------------------------------------
Summary: 100 repos scanned, 25 with issues
  - Uncommitted changes: 20
  - Replace directives:  5
  - Module mismatches:   3
```

### Table Format (`-format table`)

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
gitscan -dep github.com/grokify/gogithub ~/go/src/github.com/grokify

# Include nested go.mod files (monorepos, nested modules)
gitscan -dep github.com/grokify/gogithub -recurse ~/go/src/github.com/grokify

# Output as table
gitscan -dep github.com/grokify/mogo -format table ~/go/src/github.com/grokify
```

### Dependency Search Output

When using `-dep`, only repos containing that dependency are shown:

```
[DEP] my-service
    - Module: github.com/grokify/my-service

[DEP] another-project
    - Module: github.com/grokify/another-project

----------------------------------------
Summary: 100 repos scanned, 2 depend on github.com/grokify/gogithub
```

With `-recurse`, nested modules are also checked and displayed:

```
[DEP] monorepo
    - Module: github.com/grokify/monorepo
    - Nested modules:
      - github.com/grokify/monorepo/cmd/cli (cmd/cli/go.mod)
      - github.com/grokify/monorepo/pkg/util (pkg/util/go.mod)
```

## Use Cases

- **Pre-push audit**: Identify repos with uncommitted work before leaving for vacation
- **Dependency cleanup**: Find repos with local `replace` directives that need resolution
- **Repo hygiene**: Detect copied/renamed repos with mismatched module names
- **Breaking changes**: Find all repos to update before releasing library changes
- **Security patches**: Locate repos using vulnerable dependencies
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
