# makefmt — GNU Make Formatter & Linter

## Motivation

Existing Make linters and formatters are almost exclusively written in Python
(e.g., `checkmake`'s spiritual predecessors, various `pylint`-style Makefile
checkers). This creates friction in CI pipelines and editor integrations where a
single static binary is vastly preferable. Go solves this: one `go install` or a
single binary drop gives you a formatter that works everywhere — GitHub Actions
runners, pre-commit hooks, Neovim via `conform.nvim`, and air-gapped
environments — with zero runtime dependencies.

`makefmt` follows the UX patterns established by `gofmt`, `yamlfmt`, and
`terraform fmt`: opinionated defaults, minimal configuration, and a CLI
interface that composes well with Unix tooling and editor plugins.

---

## Goals

**MVP (v0.1)**

- Format GNU Makefiles with configurable rules.
- `--check` flag: exit 0 if already formatted, exit 1 otherwise (no output on
  success).
- `--diff` flag: print a unified diff of what would change.
- Config file discovery: `makefmt.yml`, `makefmt.yaml`, `.makefmt.yml`,
  `.makefmt.yaml` in the working directory, or explicit `--config <path>`.
- stdin/stdout support for editor piping.

**Post-MVP (v0.2+)**

- `--lint` flag: run lint rules and report diagnostics.
- Lint-specific config section with per-rule severity (`error`, `warn`, `off`).
- `--fix` flag: auto-fix lint violations where possible.

---

## CLI Interface

```
makefmt [flags] [files...]
```

When no files are given, `makefmt` reads from stdin and writes to stdout (format
mode) or reports to stderr (check/lint mode).

### Flags

| Flag               | Description                                                                                |
| ------------------ | ------------------------------------------------------------------------------------------ |
| _(none)_           | Format files in-place (default behavior).                                                  |
| `--check`          | Exit 1 if any file is not formatted. No modifications.                                     |
| `--diff`           | Print unified diff of formatting changes to stdout. No modifications.                      |
| `--lint`           | Run lint rules instead of formatting. Reports diagnostics to stderr.                       |
| `--config <path>`  | Explicit path to config file. Overrides discovery.                                         |
| `--write` / `-w`   | Write result to file (default when files are passed, required to disambiguate with stdin). |
| `--quiet` / `-q`   | Suppress informational output. Only errors and `--diff` output.                            |
| `--verbose` / `-v` | Print files as they are processed.                                                         |
| `--version`        | Print version and exit.                                                                    |

### Exit Codes

| Code | Meaning                                                                   |
| ---- | ------------------------------------------------------------------------- |
| 0    | Success. All files formatted / no lint violations.                        |
| 1    | Formatting diff detected (`--check`) or lint violations found (`--lint`). |
| 2    | Usage error, bad config, I/O failure.                                     |

### Examples

```bash
# Format a file in-place
makefmt Makefile

# Check CI — fails if anything would change
makefmt --check Makefile

# See what would change
makefmt --diff Makefile

# Format from stdin (editor integration)
cat Makefile | makefmt

# Lint (post-MVP)
makefmt --lint Makefile

# Explicit config
makefmt --config ~/shared/makefmt.yml Makefile
```

---

## Config File

Discovery order (first match wins):

1. `--config <path>` (if provided)
2. `makefmt.yml`
3. `makefmt.yaml`
4. `.makefmt.yml`
5. `.makefmt.yaml`

All paths are relative to the current working directory. If no config is found,
built-in defaults apply.

### Schema

```yaml
# makefmt.yml

formatter:
  # Indentation for recipe lines. Makefiles require tabs for recipes,
  # but this controls the width for alignment/display purposes.
  indent_style: tab # "tab" (only valid value for recipes per POSIX)
  tab_width: 4 # tab display width used for alignment decisions

  # Blank line normalization
  max_blank_lines: 2 # collapse runs of blank lines to at most N
  insert_final_newline: true # ensure file ends with a newline
  trim_trailing_whitespace: true

  # Variable assignment alignment
  align_assignments: false # align '=' across consecutive assignments
  assignment_spacing: space # "space" → VAR = val, "no_space" → VAR=val, "preserve"

  # Target formatting
  sort_prerequisites: false # alphabetically sort prerequisites
  align_backslash_continuations: true # vertically align trailing backslashes
  backslash_column: 79 # column for aligned backslashes (0 = auto)

  # Comment formatting
  space_after_comment:
    true # enforce "# comment" not "#comment"
    # does NOT touch ##, ##@, banners, or inline ## help

  # Conditional / include formatting
  indent_conditionals: true # indent bodies of ifeq/ifdef/etc.
  conditional_indent: 2 # number of spaces for conditional indentation

  # Recipe formatting
  recipe_prefix:
    preserve # "preserve" keeps @ and @-space as-is
    # "at" normalizes to @cmd, "at_space" to @ cmd

# Post-MVP: lint rules
lint:
  rules:
    no-tabs-in-spaces-context: error # tabs where spaces expected
    recipe-must-use-tab: error # spaces in recipe lines
    no-trailing-whitespace: warn # trailing whitespace
    phony-targets-declared: warn # targets without .PHONY
    undefined-variable-reference: warn # $(VAR) with no assignment
    no-hardcoded-shell: off # /bin/bash vs $(SHELL)
    consistent-variable-style: off # $() vs ${}
    consistent-recipe-prefix: off # mixed @ vs "@ " in recipe lines

  # Ignore patterns (glob)
  exclude:
    - "vendor/**"
    - "third_party/**"
```

### Rule Interface Contract

Every formatter and lint rule implements a simple Go interface (detailed in
Architecture below). Adding a new rule means:

1. Create a file in `internal/rules/format/` or `internal/rules/lint/`.
2. Implement the interface.
3. Register it in the rule registry with a config key.

No changes to CLI code, config parsing, or the engine are needed.

---

## Architecture

The design cleanly separates parsing, rule execution, and output so that
formatting and linting share the same AST and config pipeline.

### High-Level Data Flow

```
                   ┌──────────┐
   input files ──▶ │  Parser  │──▶ AST ([]Node)
                   └──────────┘
                        │
              ┌─────────┴─────────┐
              ▼                   ▼
       ┌────────────┐     ┌────────────┐
       │ Formatter  │     │  Linter    │
       │  Engine    │     │  Engine    │
       └────────────┘     └────────────┘
              │                   │
              ▼                   ▼
        formatted text     []Diagnostic
              │                   │
    ┌─────────┼─────────┐        │
    ▼         ▼         ▼        ▼
  write    diff      check    report
```

### Package Layout

```
makefmt/
├── cmd/
│   └── makefmt/
│       └── main.go              # CLI entry point, flag parsing
├── internal/
│   ├── config/
│   │   ├── config.go            # Config struct, defaults
│   │   └── loader.go            # File discovery, YAML unmarshalling
│   ├── parser/
│   │   ├── lexer.go             # Tokenizer for Makefile syntax
│   │   ├── ast.go               # Node types: Comment, Assignment, Rule, Recipe,
│   │   │                        #   Conditional, Include, Directive, BlankLine
│   │   └── parser.go            # Token stream → AST
│   ├── formatter/
│   │   ├── engine.go            # Walks AST, applies FormatRules in order
│   │   ├── rule.go              # FormatRule interface
│   │   └── writer.go            # AST → formatted text output
│   ├── linter/
│   │   ├── engine.go            # Walks AST, collects Diagnostics
│   │   ├── rule.go              # LintRule interface
│   │   └── diagnostic.go        # Diagnostic type (file, line, col, severity, msg)
│   ├── rules/
│   │   ├── format/
│   │   │   ├── trailing_whitespace.go
│   │   │   ├── blank_lines.go
│   │   │   ├── final_newline.go
│   │   │   ├── backslash_align.go
│   │   │   ├── assignment_spacing.go
│   │   │   ├── comment_spacing.go
│   │   │   └── conditional_indent.go
│   │   ├── lint/
│   │   │   ├── recipe_tab.go
│   │   │   ├── phony_declared.go
│   │   │   └── undefined_var.go
│   │   └── registry.go          # Maps config keys → rule constructors
│   └── runner/
│       └── runner.go            # Orchestrates: parse → format/lint → output/check/diff
├── pkg/
│   └── diff/
│       └── diff.go              # Unified diff generation (public, reusable)
├── testdata/
│   ├── basic/
│   │   ├── input.mk
│   │   └── expected.mk
│   ├── continuations/
│   │   ├── input.mk
│   │   └── expected.mk
│   └── ...
├── go.mod
├── go.sum
├── Makefile
└── makefmt.yml                  # Dogfood config
```

### Core Interfaces

```go
// internal/parser/ast.go

type NodeType int

const (
    NodeComment NodeType = iota
    NodeSectionHeader    // ##@ Section Name (self-documenting help headers)
    NodeBannerComment    // Decorative separators (###..., # ===..., ## box ##)
    NodeBlankLine
    NodeAssignment       // VAR = value, VAR := value, etc.
    NodeRule              // target: prerequisites
    NodeRecipe            // \t command (recipe line)
    NodeConditional       // ifeq/ifdef/ifndef/else/endif
    NodeInclude           // include, -include, sinclude
    NodeDirective         // .PHONY, .DEFAULT_GOAL, export, unexport, etc.
    NodeRaw               // unparseable lines preserved verbatim (incl. define/endef)
)

type Node struct {
    Type        NodeType
    Line        int        // 1-indexed source line number
    Raw         string     // original text (for diffing)
    Children    []*Node    // recipe lines under a rule, body of conditional
    Fields      NodeFields // type-specific parsed data
}

type NodeFields struct {
    // Assignment
    VarName     string
    AssignOp    string     // =, :=, ::=, ?=, +=, !=
    VarValue    string

    // Rule
    Targets     []string
    Prerequisites []string
    OrderOnly   []string   // after |
    InlineHelp  string     // "## Description" trailing comment on rule lines

    // Conditional
    Directive   string     // ifeq, ifdef, ifndef, else, endif
    Condition   string     // the condition expression

    // Include
    IncludeType string     // include, -include, sinclude
    Paths       []string

    // Comment / SectionHeader / BannerComment
    Text        string
    Inline      bool       // trailing comment on another line
    Prefix      string     // "#", "##", "##@" — preserved exactly by the writer
}
```

```go
// internal/formatter/rule.go

// FormatRule transforms AST nodes. Rules are applied in registered order.
type FormatRule interface {
    // Name returns the config key for this rule (e.g., "trim_trailing_whitespace").
    Name() string

    // Format receives the full AST and config, returns a modified AST.
    // Rules should not mutate the input; return new nodes where changes are needed.
    Format(nodes []*parser.Node, cfg *config.FormatterConfig) []*parser.Node
}
```

```go
// internal/linter/rule.go

type Severity int

const (
    SeverityOff Severity = iota
    SeverityWarn
    SeverityError
)

// LintRule inspects AST nodes and produces diagnostics.
type LintRule interface {
    // Name returns the config key for this rule.
    Name() string

    // DefaultSeverity returns the severity when not configured.
    DefaultSeverity() Severity

    // Check runs the rule against the AST and returns any violations.
    Check(nodes []*parser.Node, cfg *config.LintConfig) []Diagnostic
}
```

```go
// internal/rules/registry.go

// Registry maps config keys to rule constructors.
// Adding a rule = one Register() call.
var formatRules []FormatRule
var lintRules   []LintRule

func RegisterFormatRule(r FormatRule) { formatRules = append(formatRules, r) }
func RegisterLintRule(r LintRule)     { lintRules = append(lintRules, r) }

func init() {
    RegisterFormatRule(&format.TrailingWhitespace{})
    RegisterFormatRule(&format.BlankLines{})
    RegisterFormatRule(&format.FinalNewline{})
    RegisterFormatRule(&format.BackslashAlign{})
    RegisterFormatRule(&format.AssignmentSpacing{})
    RegisterFormatRule(&format.CommentSpacing{})
    RegisterFormatRule(&format.ConditionalIndent{})

    RegisterLintRule(&lint.RecipeTab{})
    RegisterLintRule(&lint.PhonyDeclared{})
    RegisterLintRule(&lint.UndefinedVar{})
}
```

### Runner Orchestration

```go
// internal/runner/runner.go (simplified)

func Run(opts Options) int {
    cfg := config.Load(opts.ConfigPath)
    exitCode := 0

    for _, path := range opts.Files {
        src := readFile(path)                  // or stdin
        nodes := parser.Parse(src)

        if opts.Lint {
            diags := linter.Run(nodes, cfg.Lint)
            reportDiagnostics(diags)
            if hasErrors(diags) { exitCode = 1 }
            continue
        }

        formatted := formatter.Run(nodes, cfg.Formatter)
        output := writer.Write(formatted)

        switch {
        case opts.Check:
            if output != src { exitCode = 1 }
        case opts.Diff:
            d := diff.Unified(path, src, output)
            if d != "" {
                fmt.Print(d)
                exitCode = 1
            }
        default:
            writeFile(path, output)            // or stdout
        }
    }
    return exitCode
}
```

---

## Parsing Strategy

GNU Make syntax is notoriously context-sensitive. Rather than attempting a full
grammar, the parser operates line-by-line with minimal state tracking.

### Line Classification

Each line is classified in order of precedence:

1. **Section Header** — starts with `##@` (the self-documenting help system
   pattern).
2. **Banner Comment** — a comment line consisting entirely of repeated
   characters (`#`, `=`, `-`) or box-style patterns (e.g., `## Title ##`).
   Detected by regex: line matches `^#+$`, `^# [=\-#]{3,}`, or
   `^#{2,}\s.*\s#{2,}$`.
3. **Comment** — starts with `#` (after optional whitespace). Preserves prefix
   (`#`, `##`) in the AST.
4. **Recipe** — starts with a tab _and_ follows a rule or another recipe line.
5. **Conditional** — starts with `ifeq`, `ifneq`, `ifdef`, `ifndef`, `else`,
   `endif` (after optional whitespace).
6. **Include** — starts with `include`, `-include`, or `sinclude`.
7. **Assignment** — contains `=`, `:=`, `::=`, `?=`, `+=`, or `!=` outside of a
   rule context.
8. **Rule** — contains `:` with target pattern (not `::=`). If the line also
   contains `## text` after the prerequisites, the `InlineHelp` field captures
   it.
9. **Directive** — starts with `.PHONY`, `.DEFAULT_GOAL`, `export`, `unexport`,
   `vpath`, `override`, etc.
10. **Blank** — empty or whitespace-only.
11. **Raw** — anything else, including `define`/`endef` blocks (preserved
    verbatim).

### Continuation Lines

Lines ending in `\` are joined with their continuation before classification.
The parser tracks the original line boundaries so formatting can re-wrap and
align backslashes.

### State Machine

The parser maintains a small state stack to handle:

- Whether we are "inside a rule" (so tab-indented lines are recipes, not
  errors). Triggered by both explicit targets (`build:`) and pattern rules
  (`log-%:`, `%:`).
- Conditional nesting depth (for `indent_conditionals`).
- Whether we are inside a `define ... endef` block (all content treated as
  `NodeRaw`, preserved verbatim).
- Whether the previous non-blank line was an assignment (for `align_assignments`
  grouping).

Pattern rules like `log-%:` and `%:` are parsed as `NodeRule` with the `%`
preserved in the `Targets` field. The recipe lines under them follow the same
rules as any other target.

---

## MVP Formatting Rules

These are the rules included in v0.1, each mapped to a config key:

### `trim_trailing_whitespace`

Remove trailing spaces and tabs from every line.

### `insert_final_newline`

Ensure the file ends with exactly one newline character.

### `max_blank_lines`

Collapse consecutive blank lines down to the configured maximum.

### `assignment_spacing`

Normalize whitespace around assignment operators. `space` mode ensures
`VAR = val`, `no_space` ensures `VAR=val`, `preserve` leaves as-is.

### `align_backslash_continuations`

Align trailing backslashes in continuation blocks to a consistent column
(configurable or auto-detected from the longest line).

### `space_after_comment`

Ensure `#` is followed by a space (e.g., `# comment` not `#comment`).
Exceptions: shebangs (`#!`), section headers (`##@`), banner lines (`####...`),
and inline help comments (`## Description` on rule lines) are all preserved
exactly as-is. The rule only normalizes standalone single-`#` comments.

### `preserve_banner_comments`

Banner comments (`# ====...`, `########...`, `## Box Title ##`) and section
headers (`##@`) are preserved verbatim — no whitespace normalization, no
reformatting. These nodes pass through the formatter untouched.

### `indent_conditionals`

Indent the body of `ifeq`/`ifdef`/`ifndef` blocks by the configured number of
spaces. `else` and `endif` align with the opening directive.

---

## Post-MVP: Linting

The linting engine shares the same AST but produces diagnostics instead of
transformed output. Each lint rule reports violations with file, line, column,
severity, and a human-readable message.

### Planned Lint Rules

| Rule                           | Default | Description                                                 |
| ------------------------------ | ------- | ----------------------------------------------------------- |
| `recipe-must-use-tab`          | error   | Recipe lines must start with a tab, not spaces.             |
| `no-tabs-in-spaces-context`    | error   | Tabs in non-recipe contexts where spaces are expected.      |
| `no-trailing-whitespace`       | warn    | Trailing whitespace on any line.                            |
| `phony-targets-declared`       | warn    | Targets that look phony (no file output) but lack `.PHONY`. |
| `undefined-variable-reference` | warn    | `$(VAR)` or `${VAR}` with no visible assignment.            |
| `no-hardcoded-shell`           | off     | Direct `/bin/bash` or `/bin/sh` instead of `$(SHELL)`.      |
| `consistent-variable-style`    | off     | Mixed `$()` and `${}` syntax.                               |

### Diagnostic Output Format

```
Makefile:12:1: error: recipe line uses spaces instead of tab (recipe-must-use-tab)
Makefile:24:0: warn: target 'clean' is not declared .PHONY (phony-targets-declared)
```

Compatible with the `errorformat` used by Vim/Neovim quickfix and `conform.nvim`
diagnostics.

---

## Editor Integration

### Neovim / LazyVim with conform.nvim

```lua
-- lua/plugins/conform.lua
return {
  "stevearc/conform.nvim",
  opts = {
    formatters_by_ft = {
      make = { "makefmt" },
    },
    formatters = {
      makefmt = {
        command = "makefmt",
        stdin = true,
      },
    },
  },
}
```

### Neovim Diagnostics (post-MVP, via nvim-lint or efm)

```lua
-- nvim-lint config
require("lint").linters.makefmt = {
  cmd = "makefmt",
  args = { "--lint" },
  stdin = true,
  stream = "stderr",
  parser = require("lint.parser").from_errorformat("%f:%l:%c: %t%*[^:]: %m"),
}
```

### Pre-commit Hook

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/<org>/makefmt
    rev: v0.1.0
    hooks:
      - id: makefmt
        name: makefmt
        entry: makefmt --check
        language: golang
        files: '(^|/)([Mm]akefile|.*\.mk)$'
```

### GitHub Actions

```yaml
- name: Check Makefile formatting
  run: |
    go install github.com/<org>/makefmt/cmd/makefmt@latest
    makefmt --check Makefile
```

---

## Testing Strategy

### Golden File Tests

Each rule has a `testdata/<rule>/` directory containing `input.mk` and
`expected.mk` pairs. The test harness parses `input.mk`, applies the rule, and
diffs against `expected.mk`.

A shared test helper lives in `internal/testutil/golden.go` and provides:

- `RunGolden(t, dir, cfg, formatFunc)` — run a single golden file test
- `RunGoldenDir(t, testdataDir, cfg, formatFunc)` — auto-discover and run all
  subdirectories under a testdata path

Golden files can be regenerated from current formatter output using the
`-update` flag:

```bash
go test ./... -update     # regenerate all expected.mk files from current output
```

When `-update` is passed, the test helper writes the actual formatter output to
`expected.mk` instead of comparing against it. This is the standard pattern
used by Go formatting tools:

- **gofumpt** (`github.com/mvdan/gofumpt`) — uses `testscript` with txtar
  archives in `format/format_test.go`; supports `-update` to rewrite archives.
- **yamlfmt** (`github.com/google/yamlfmt`) — uses `before.yaml` / `after.yaml`
  pairs in `formatters/basic/basic_test.go` with directory-per-feature layout.
- **shfmt** (`github.com/mvdan/sh`) — uses `.in` / `.out` file pairs in
  `syntax/printer_test.go` with `filepath.Walk` discovery.

Example golden file pair — `testdata/full_format/input.mk`:

```makefile
# Project Variables

PROJECT_NAME := my-project
PROJECT_OWNER:=donaldgifford
DESCRIPTION:= A project

## Go Variables

GO ?= go
GO_PACKAGE:= github.com/$(PROJECT_OWNER)/$(PROJECT_NAME)

###############
##@ Development

.PHONY: build test lint

build: ## Build the binary
	@ $(MAKE) --no-print-directory log-$@
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(PROJECT_NAME) ./cmd/$(PROJECT_NAME)

test: ## Run tests
	@ $(MAKE) --no-print-directory log-$@
	@go test -v -race ./...


release: ## Create release (use with TAG=v1.0.0)
	@ $(MAKE) --no-print-directory log-$@
	@if [ -z "$(TAG)" ]; then \
		echo "Error: TAG is required"; \
			exit 1; \
	fi
	git tag -a $(TAG) -m "Release $(TAG)"



########
##@ Help

log-%:
	@grep -h -E '^$*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN { FS = ":.*?## " }; { printf "\033[36m==> %s\033[0m\n", $$2 }'
```

Expected output — `testdata/full_format/expected.mk` (with `max_blank_lines: 2`,
`assignment_spacing: space`, `align_assignments: false`):

```makefile
# Project Variables

PROJECT_NAME := my-project
PROJECT_OWNER := donaldgifford
DESCRIPTION := A project

## Go Variables

GO ?= go
GO_PACKAGE := github.com/$(PROJECT_OWNER)/$(PROJECT_NAME)

###############
##@ Development

.PHONY: build test lint

build: ## Build the binary
	@ $(MAKE) --no-print-directory log-$@
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(PROJECT_NAME) ./cmd/$(PROJECT_NAME)

test: ## Run tests
	@ $(MAKE) --no-print-directory log-$@
	@go test -v -race ./...


release: ## Create release (use with TAG=v1.0.0)
	@ $(MAKE) --no-print-directory log-$@
	@if [ -z "$(TAG)" ]; then \
		echo "Error: TAG is required"; \
		exit 1; \
	fi
	git tag -a $(TAG) -m "Release $(TAG)"


########
##@ Help

log-%:
	@grep -h -E '^$*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN { FS = ":.*?## " }; { printf "\033[36m==> %s\033[0m\n", $$2 }'
```

Key things this test validates: assignment operator spacing is normalized
(`PROJECT_OWNER:=` → `PROJECT_OWNER :=`), triple blank lines are collapsed to
double, `##@` section headers are untouched, `@ $(MAKE)` recipe prefix is
preserved, banner comments (`###...`, `########`) are preserved, backslash
continuation alignment is normalized, and pattern rules (`log-%:`) parse
correctly.

### Table-Driven Unit Tests

Parser and individual rule logic use standard Go table-driven tests for edge
cases: empty files, continuation-only files, deeply nested conditionals, mixed
tabs/spaces, `define`/`endef` blocks, etc.

### Integration Tests

End-to-end tests invoke the `makefmt` binary with various flag combinations and
assert on exit codes, stdout, and stderr.

### Fuzz Testing

The parser is fuzz-tested with `go test -fuzz` to catch panics or infinite loops
on malformed input.

### Dogfooding

The project's own `Makefile` is formatted by `makefmt` and checked in CI.

---

## Milestones

### M1 — Parser + Core Formatting (weeks 1–3)

- Lexer and line-based parser with continuation handling.
- AST node types for all Makefile constructs.
- MVP formatting rules: trailing whitespace, blank lines, final newline,
  assignment spacing.
- Writer that serializes AST back to text.
- Golden file test harness.

### M2 — CLI + Config + Diff (weeks 3–4)

- CLI with `--check`, `--diff`, `--write`, stdin/stdout support.
- Config file discovery and YAML loading.
- Unified diff output.
- Exit code semantics.

### M3 — Advanced Formatting (weeks 4–5)

- Backslash continuation alignment.
- Conditional indentation.
- Comment spacing.
- `align_assignments` (optional).

### M4 — Polish + Release v0.1 (week 6)

- Integration tests, fuzz tests.
- Editor integration docs (conform.nvim, pre-commit).
- README, `--help` text, man page.
- Goreleaser config for cross-platform binaries.
- Tag v0.1.0.

### M5 — Linting Engine + Rules (post-MVP)

- Lint engine with diagnostic collection.
- Lint config section with per-rule severity.
- Initial lint rules: `recipe-must-use-tab`, `phony-targets-declared`,
  `no-trailing-whitespace`.
- `--lint` flag integration.
- `--fix` flag for auto-fixable lint violations.

---

## Resolved Design Decisions

Based on analysis of real-world Makefile patterns (single-file projects,
multi-file domain-split repos with `include`), the following questions are
resolved:

### 1. `define` / `endef` blocks → Opaque for MVP

Not present in any of the reference Makefiles. Treat `define`/`endef` as opaque
raw blocks — preserve content verbatim, only apply `trim_trailing_whitespace`
and `insert_final_newline`. Revisit in post-MVP if demand arises.

### 2. GNU Make extensions → Parse, don't format internals

The reference files make heavy use of GNU extensions: `$(shell ...)`, `?=`,
`$(MAKE)`, `$(MAKEFILE_LIST)`, `$(CMD)`, etc. The parser must handle all GNU
Make syntax without breaking. Formatting rules operate on the structural level
(assignment alignment, whitespace) and do not attempt to rewrite the insides of
`$(...)` expressions.

### 3. Tab handling in alignment → Scoped to variable blocks only

`align_assignments` only applies within consecutive assignment blocks (lines
containing `=`, `:=`, `?=`, `+=`). Recipe lines are never touched by this rule.
Assignment blocks are separated by blank lines, comments, or any non-assignment
line — this naturally creates the boundary. Alignment pads with spaces between
the variable name and the operator, which is the existing convention in the
reference files:

```makefile
# These are already aligned — the formatter preserves/enforces this:
PROJECT_NAME  := zfs_exporter
PROJECT_OWNER := donaldgifford
DESCRIPTION   := Prometheus exporter for ZFS
```

### 4. Parallel safe writes → Sequential for MVP, concurrent post-MVP

The reference repos have at most ~6 included Makefiles. Sequential processing is
fine for MVP. Post-MVP can add `errgroup`-based concurrency for monorepo use
cases.

---

## Observations from Real Makefiles

These patterns were extracted from the reference files and directly inform
parser design and formatting rules.

### Self-documenting help system (`##@` and `##`)

This is the dominant pattern. Targets use `## Description` for inline help, and
`##@` creates section headers in the help output:

```makefile
##@ Go Development      ← section header (parsed by awk in the help target)

build: build-core ## Build everything (core)   ← inline help comment
```

The parser must distinguish three comment types:

- **Standalone comments** — `# text` on their own line
- **Section headers** — `##@ Section Name` (special `##@` prefix)
- **Inline help comments** — `## Description` trailing a rule line

The formatter must preserve `##@` and `##` prefixes exactly. The
`space_after_comment` rule should handle `#` → `#` but leave `##` and `##@`
alone (they already have intentional formatting).

### Grouped `.PHONY` declarations at section tops

```makefile
.PHONY: build build-core
.PHONY: test test-all test-pkg test-report test-coverage
.PHONY: lint lint-fix fmt clean generate mocks
```

Multiple `.PHONY` lines are grouped at the top of logical sections, not
co-located with each target. The formatter should not reorder or merge these —
they are intentionally grouped for readability.

### `@ $(MAKE)` logging pattern

Nearly every target starts with:

```makefile
target: ## Description
	@ $(MAKE) --no-print-directory log-$@
```

The `@` prefix (with a space after `@`) suppresses output. This is intentional
— not a whitespace error. The formatter must preserve `@` vs `@` in recipe
lines. This also means a lint rule for "inconsistent `@` usage" could be
valuable post-MVP.

### Pattern rules (`log-%`)

```makefile
log-%:
	@grep -h -E '^$*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN { FS = ":.*?## " }; { printf "\033[36m==> %s\033[0m\n", $$2 }'
```

Pattern rules with `%` must be parsed as valid rule targets, not treated as
errors. The `$$` in recipe lines (escaped `$` for shell) must be preserved
verbatim.

### Banner/separator comments

```makefile
# =============================================================================
# Release Targets
# =============================================================================

########################################################################
## Self-Documenting Makefile Help                                     ##
########################################################################
```

Decorative comment blocks using `=`, `#`, or box-drawing patterns. The formatter
should preserve these exactly — they are intentional visual structure. A
potential post-MVP formatting rule could normalize separator styles, but MVP
treats them as raw comments.

### Multi-file `include` structure

```makefile
include scripts/makefiles/common.mk
include scripts/makefiles/go.mk
# include scripts/makefiles/docker.mk    ← commented-out includes
```

The formatter processes each file independently — it does not follow `include`
directives. Variable references like `$(BIN_DIR)` that are defined in
`common.mk` but used in `go.mk` are not resolved. This is correct for a
formatter; lint rules that check for undefined variables (post-MVP) will need a
`--include-path` flag or similar mechanism.

### Catch-all pattern rule

```makefile
%:
	@:
```

Used at the end of the top-level Makefile to silently swallow extra arguments
(enables `make adr "My Title"`). The parser must handle `%:` as a valid rule
target.

### Double blank lines are occasional

Some files have double blank lines between sections (e.g., after
`test-coverage`, after `release-local`). The `max_blank_lines: 2` default
accommodates this. Setting it to `1` would normalize more aggressively.

---

## Remaining Notes

1. **Commented-out `include` directives** — Treated as plain comments. In
   practice, unused includes get deleted rather than commented out. No special
   handling needed.
2. **`@` vs `@` normalization** — Default is `preserve`. Neither form affects
   Makefile correctness — it's purely stylistic. Users who want consistency can
   set `recipe_prefix: at` or `recipe_prefix: at_space` in config, or use the
   `consistent-recipe-prefix` lint rule post-MVP.
3. **Alignment scope detection** — Consecutive assignment lines form a group. A
   blank line, comment, or any non-assignment line breaks the group. This keeps
   variable blocks compact and readable without over-reaching into unrelated
   sections.
