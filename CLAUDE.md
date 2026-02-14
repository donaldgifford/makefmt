# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

makefmt is a fast, opinionated GNU Make formatter written in Go. Single binary, zero dependencies. Designed for format-on-save (conform.nvim) and CI checks. Follows the UX patterns of `gofmt`, `yamlfmt`, and `terraform fmt`.

**Status**: MVP phase — design is complete in `docs/DESIGN.md`, implementation is starting from `cmd/makefmt/main.go`.

## Build & Development Commands

```bash
make build          # Build binary to build/bin/makefmt
make test           # Run all tests with race detector
make test-pkg PKG=./internal/parser  # Test a single package
make test-coverage  # Tests with coverage report
make lint           # Run golangci-lint
make lint-fix       # Lint with auto-fix
make fmt            # Format with gofmt + goimports
make ci             # Full CI pipeline: lint + test + build
make check          # Quick pre-commit: lint + test
make help           # Show all available targets
```

Tool versions are managed by mise (`mise.toml`). Run `mise install` to set up the toolchain (Go 1.25.7, golangci-lint 2.8.0, etc.).

## Architecture

The design spec is in `docs/DESIGN.md` (962 lines) — treat it as the authoritative reference for all implementation decisions.

### Data Flow

```
input files → Parser → AST ([]Node) → Formatter Engine → formatted text → write/diff/check
                                     → Linter Engine    → []Diagnostic  → report
```

### Package Layout

- `cmd/makefmt/` — CLI entry point, flag parsing (stdlib `flag`)
- `internal/config/` — Config struct, defaults, YAML file discovery and loading
- `internal/parser/` — Lexer, AST node types, line-by-line parser with state machine
- `internal/formatter/` — Engine that walks AST and applies FormatRules in order
- `internal/linter/` — Engine that walks AST and collects Diagnostics (post-MVP)
- `internal/rules/format/` — Individual formatting rule implementations
- `internal/rules/lint/` — Individual lint rule implementations
- `internal/rules/registry.go` — Maps config keys to rule constructors
- `internal/testutil/` — Shared golden file test helper (`-update` flag support)
- `internal/runner/` — Orchestration: parse → format/lint → output/check/diff
- `pkg/diff/` — Unified diff generation (public, reusable Myers diff)
- `testdata/` — Golden file tests (input.mk / expected.mk pairs)

### Key Interfaces

**FormatRule** (`internal/formatter/rule.go`): Receives full AST + config, returns modified AST. Rules must not mutate input — return new nodes. Registered via `RegisterFormatRule()` in `internal/rules/registry.go`.

**LintRule** (`internal/linter/rule.go`): Inspects AST, returns `[]Diagnostic` with severity (off/warn/error).

Adding a new rule: create file in `internal/rules/format/` or `internal/rules/lint/`, implement the interface, register in the registry. No CLI or config parsing changes needed.

### Parser Design

The parser is line-by-line with a state stack (not a full grammar). Line classification order matters: SectionHeader → BannerComment → Comment → Recipe → Conditional → Include → Assignment → Rule → Directive → Blank → Raw. Lines ending in `\` are joined before classification. `define`/`endef` blocks are preserved as NodeRaw.

### AST Node Types

NodeComment, NodeSectionHeader (`##@`), NodeBannerComment (decorative separators), NodeBlankLine, NodeAssignment, NodeRule, NodeRecipe, NodeConditional, NodeInclude, NodeDirective, NodeRaw.

## MVP Formatting Rules (v0.1)

1. `trim_trailing_whitespace`
2. `insert_final_newline`
3. `max_blank_lines` (default: 2)
4. `assignment_spacing` (space/no_space/preserve)
5. `align_backslash_continuations`
6. `space_after_comment` (skips `##`, `##@`, banners, shebangs)
7. `indent_conditionals`
8. `preserve_banner_comments`

## Config Discovery

First match wins: `--config <path>` → `makefmt.yml` → `makefmt.yaml` → `.makefmt.yml` → `.makefmt.yaml`. Falls back to built-in defaults.

## Testing Conventions

- Golden file tests in `testdata/` with `input.mk` / `expected.mk` pairs
- Table-driven tests for parser and individual rules
- Race detector enabled by default (`-race`)
- Golangci-lint config in `.golangci.yml` (40+ linters, Uber Go Style Guide based)

## Go Module

`github.com/donaldgifford/makefmt` — goimports local grouping: `github.com/donaldgifford`

## CLI

```
makefmt [flags] [files...]
```

Reads stdin when no files given. Key flags: `--check` (exit 1 if unformatted), `--diff` (unified diff), `--write`/`-w`, `--config <path>`. Exit codes: 0 success, 1 formatting needed, 2 usage/IO error.
