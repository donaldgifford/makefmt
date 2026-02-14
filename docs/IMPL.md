# makefmt Implementation Plan

This document is the phased implementation plan for makefmt v0.1.0. Each phase
lists its tasks (checkboxes), files to create or modify, success criteria, and
dependencies on prior phases.

The authoritative design reference is `docs/DESIGN.md`. This plan does not
repeat design rationale — it specifies *what to build and in what order*.

---

## Dependency Graph

```
Phase 0  (template fixes)       — no deps
Phase 1  (AST + parser)         — no deps
Phase 2  (writer)               — depends on Phase 1
Phase 3  (formatter engine)     — depends on Phase 1, Phase 2
Phase 4  (config)               — no deps (types only; wired in Phase 5)
Phase 5  (CLI + runner + diff)  — depends on Phase 2, Phase 3, Phase 4
Phase 6  (advanced rules)       — depends on Phase 3
Phase 7  (polish + release)     — depends on all above
Phase 8  (linter, post-MVP)     — depends on Phase 1, Phase 4
```

---

## Phase 0 — Fix Template Artifacts

The repo was scaffolded from a project template (`forge`). Several config files
still reference the wrong project name, binary, or module path.

### Tasks

- [x] **`.goreleaser.yml`** — Replace all `forge` references with `makefmt`:
  - `builds[0].id`: `forge` → `makefmt`
  - `builds[0].binary`: `forge` → `makefmt`
  - `builds[0].main`: `./cmd/forge` → `./cmd/makefmt`
  - `release.github.name`: `forge` → `makefmt`
  - `release.header`: `tar -xzf foreg_*.tar.gz` → `tar -xzf makefmt_*.tar.gz`
  - `release.header`: `mv forge ~/.local/bin/` → `mv makefmt ~/.local/bin/`
  - `release.footer`: `github.com/donaldgifford/forge/compare/` → `github.com/donaldgifford/makefmt/compare/`
- [x] **`.golangci.yml:273-274`** — Fix goimports local-prefixes:
  - `github.com/donaldgifford/keycloak-cli` → `github.com/donaldgifford/makefmt`
- [x] **`.github/workflows/ci.yml:58`** — Fix codecov slug:
  - `slug: donaldgifford/` → `slug: donaldgifford/makefmt`
- [x] **`.golangci.yml:77`** — Decide on `goheader` linter:
  - Either add a license header template or disable the `goheader` linter
  - Recommendation: disable `goheader` for now (no license header convention established)

### Files Modified

- `.goreleaser.yml`
- `.golangci.yml`
- `.github/workflows/ci.yml`

### Success Criteria

- `goreleaser check` passes (no schema errors)
- `golangci-lint run ./...` does not fail on config errors
- CI workflow references the correct codecov slug

### Dependencies

None.

---

## Phase 1 — AST Node Types + Parser

Build the line-by-line parser that converts Makefile source text into an AST
(`[]*Node`). This is the foundation everything else depends on.

### Tasks

- [x] Create `internal/parser/ast.go`:
  - Define `NodeType` enum (Comment, SectionHeader, BannerComment, BlankLine, Assignment, Rule, Recipe, Conditional, Include, Directive, Raw)
  - Define `Node` struct with `Type`, `Line`, `Raw`, `Children`, `Fields`
  - Define `NodeFields` struct (VarName, AssignOp, VarValue, Targets, Prerequisites, OrderOnly, InlineHelp, Directive, Condition, IncludeType, Paths, Text, Inline, Prefix)
  - Add `Clone()` method on `*Node` (deep copy for immutable rule transforms)
- [x] Create `internal/parser/parser.go`:
  - `Parse(src string) []*Node` — main entry point
  - Line-by-line classification with the priority order from DESIGN.md: SectionHeader → BannerComment → Comment → Recipe → Conditional → Include → Assignment → Rule → Directive → Blank → Raw
  - Continuation line joining (lines ending in `\`)
  - State machine: `inRule` flag (tab lines are recipes), `inDefine` flag (verbatim until `endef`), conditional nesting depth
  - Recipe lines become `Children` of their parent `NodeRule`
  - Conditional bodies become `Children` of their parent `NodeConditional`
- [x] Create `internal/parser/parser_test.go`:
  - Table-driven tests for each node type classification
  - Edge cases: empty file, blank-only file, continuation-only lines, `define`/`endef` blocks, pattern rules (`%:`, `log-%:`), catch-all `%:`, deeply nested conditionals, `##@` section headers, banner comments, inline help `## Description` on rule lines, mixed assignment operators
  - Test that `Raw` field preserves original text for every node

### Files Created

- `internal/parser/ast.go`
- `internal/parser/parser.go`
- `internal/parser/parser_test.go`

### Success Criteria

- `go test -race ./internal/parser/...` passes
- Parser correctly classifies all line types from the example Makefiles in `docs/examples/`
- Continuation lines are joined; original line numbers are preserved
- `define`/`endef` blocks produce `NodeRaw` with verbatim content
- Pattern rules (`%:`, `log-%:`) produce `NodeRule`

### Dependencies

None.

---

## Phase 2 — Writer (AST → Text)

The writer serializes an AST back into Makefile text. It is the inverse of the
parser — `writer.Write(parser.Parse(src))` should round-trip correctly for
already-formatted input.

### Tasks

- [x] Create `internal/formatter/writer.go`:
  - `Write(nodes []*Node) string` — walks the node list and emits text
  - Each `NodeType` has a serialization path:
    - `NodeComment`: emit `Prefix + " " + Text` (or just `Prefix` if empty text)
    - `NodeSectionHeader`: emit raw (verbatim)
    - `NodeBannerComment`: emit raw (verbatim)
    - `NodeBlankLine`: emit empty line
    - `NodeAssignment`: emit `VarName + " " + AssignOp + " " + VarValue` (spacing controlled by fields, not hardcoded)
    - `NodeRule`: emit `Targets : Prerequisites | OrderOnly` with optional `InlineHelp`
    - `NodeRecipe`: emit `\t` + recipe content (from Children of NodeRule)
    - `NodeConditional`: emit directive + condition, recurse into Children, emit `endif`/`else`
    - `NodeInclude`: emit `IncludeType + " " + Paths`
    - `NodeDirective`: emit raw
    - `NodeRaw`: emit raw (verbatim)
  - Ensure final output respects node ordering faithfully
- [x] Create `internal/formatter/writer_test.go`:
  - Round-trip tests: `Parse(src)` → `Write()` produces the original text for clean input
  - Test each node type serialization individually

### Files Created

- `internal/formatter/writer.go`
- `internal/formatter/writer_test.go`

### Success Criteria

- `go test -race ./internal/formatter/...` passes
- Round-trip: `Write(Parse(src)) == src` for already-formatted Makefiles
- All node types serialize correctly

### Dependencies

Phase 1 (parser/AST types).

---

## Phase 3 — Formatter Engine + Simple Rules

Build the formatter engine that applies `FormatRule` implementations in
registered order, plus the four simplest formatting rules.

### Tasks

- [x] Create `internal/formatter/rule.go`:
  - Define `FormatRule` interface: `Name() string`, `Format(nodes []*Node, cfg *config.FormatterConfig) []*Node`
- [x] Create `internal/formatter/engine.go`:
  - `Run(nodes []*Node, cfg *config.FormatterConfig, rules []FormatRule) []*Node`
  - Applies each rule in order, passing the output of one as input to the next
- [x] Create `internal/rules/registry.go`:
  - `var formatRules []FormatRule`
  - `RegisterFormatRule(r FormatRule)`
  - `FormatRules() []FormatRule` — returns registered rules in order
  - `init()` function registers all rules in execution order
- [x] Implement `internal/rules/format/trailing_whitespace.go`:
  - Rule: `trim_trailing_whitespace`
  - Strip trailing spaces/tabs from every node's text content
  - Operates on `Raw` field and all text-bearing `NodeFields`
- [x] Implement `internal/rules/format/final_newline.go`:
  - Rule: `insert_final_newline`
  - Ensure the AST ends with exactly one blank line (or the writer appends `\n`)
- [x] Implement `internal/rules/format/blank_lines.go`:
  - Rule: `max_blank_lines`
  - Collapse consecutive `NodeBlankLine` runs to at most `cfg.MaxBlankLines` (default: 2)
- [x] Implement `internal/rules/format/assignment_spacing.go`:
  - Rule: `assignment_spacing`
  - `space` mode: normalize `NodeAssignment` fields so writer emits `VAR = val` (single space around operator)
  - `no_space` mode: `VAR=val`
  - `preserve` mode: no-op
  - Handle all operator types: `=`, `:=`, `::=`, `?=`, `+=`, `!=`
- [x] Create tests for each rule:
  - `internal/rules/format/trailing_whitespace_test.go`
  - `internal/rules/format/final_newline_test.go`
  - `internal/rules/format/blank_lines_test.go`
  - `internal/rules/format/assignment_spacing_test.go`
- [x] Create golden file test harness in `internal/testutil/golden.go`:
  - `var Update = flag.Bool("update", false, "update golden files")`
  - `RunGolden(t, dir, cfg, formatFunc)` — single golden test:
    1. Read `input.mk` from dir
    2. Run formatFunc to produce actual output
    3. If `-update` flag: write actual output to `expected.mk` and return
    4. Otherwise: read `expected.mk`, compare, fail with diff on mismatch
  - `RunGoldenDir(t, testdataDir, cfg, formatFunc)` — walk all subdirectories, call `RunGolden` for each as a `t.Run` subtest
  - Usage: `go test ./... -update` regenerates all `expected.mk` files from current formatter output
  - This follows the established pattern from gofumpt, yamlfmt, and shfmt (see DESIGN.md Testing Strategy)
- [x] Create initial golden file pairs:
  - `testdata/trailing_whitespace/input.mk` + `expected.mk`
  - `testdata/blank_lines/input.mk` + `expected.mk`
  - `testdata/assignment_spacing/input.mk` + `expected.mk`

### Rule Execution Order

The registry `init()` registers rules in this order (matches DESIGN.md):

1. `trim_trailing_whitespace`
2. `insert_final_newline`
3. `max_blank_lines`
4. `assignment_spacing`
5. `align_backslash_continuations` *(Phase 6)*
6. `space_after_comment` *(Phase 6)*
7. `indent_conditionals` *(Phase 6)*
8. `preserve_banner_comments` *(Phase 6)*

Rules 5-8 are registered in Phase 6 but slots are reserved in order.

### Files Created

- `internal/formatter/rule.go`
- `internal/formatter/engine.go`
- `internal/rules/registry.go`
- `internal/testutil/golden.go`
- `internal/rules/format/trailing_whitespace.go`
- `internal/rules/format/trailing_whitespace_test.go`
- `internal/rules/format/final_newline.go`
- `internal/rules/format/final_newline_test.go`
- `internal/rules/format/blank_lines.go`
- `internal/rules/format/blank_lines_test.go`
- `internal/rules/format/assignment_spacing.go`
- `internal/rules/format/assignment_spacing_test.go`
- `testdata/trailing_whitespace/input.mk`
- `testdata/trailing_whitespace/expected.mk`
- `testdata/blank_lines/input.mk`
- `testdata/blank_lines/expected.mk`
- `testdata/assignment_spacing/input.mk`
- `testdata/assignment_spacing/expected.mk`

### Success Criteria

- `go test -race ./internal/formatter/...` passes
- `go test -race ./internal/rules/format/...` passes
- Golden file tests pass for each rule
- Engine applies rules in registered order; output of rule N is input to rule N+1
- Rules do not mutate input nodes (return new/cloned nodes where changes are made)

### Dependencies

Phase 1 (AST types), Phase 2 (writer for golden file tests).

Note: Phase 3 initially uses a stub `config.FormatterConfig` struct with
hardcoded defaults. The real config loading comes in Phase 4.

---

## Phase 4 — Config Struct + YAML Loading

Define the configuration types and implement YAML file discovery and loading.
This phase is independent of the parser — it only defines data types and a
loader.

### Tasks

- [x] Create `internal/config/config.go`:
  - `Config` struct with `Formatter FormatterConfig` and `Lint LintConfig` fields
  - `FormatterConfig` struct with all fields from DESIGN.md schema:
    - `IndentStyle` (string, default `"tab"`)
    - `TabWidth` (int, default `4`)
    - `MaxBlankLines` (int, default `2`)
    - `InsertFinalNewline` (bool, default `true`)
    - `TrimTrailingWhitespace` (bool, default `true`)
    - `AlignAssignments` (bool, default `false`)
    - `AssignmentSpacing` (string: `"space"` | `"no_space"` | `"preserve"`, default `"space"`)
    - `SortPrerequisites` (bool, default `false`)
    - `AlignBackslashContinuations` (bool, default `true`)
    - `BackslashColumn` (int, default `79`)
    - `SpaceAfterComment` (bool, default `true`)
    - `IndentConditionals` (bool, default `true`)
    - `ConditionalIndent` (int, default `2`)
    - `RecipePrefix` (string: `"preserve"` | `"at"` | `"at_space"`, default `"preserve"`)
  - `LintConfig` struct (placeholder for post-MVP, just the type with `Rules map[string]string` and `Exclude []string`)
  - `DefaultConfig() *Config` — returns config with all defaults
- [x] Create `internal/config/loader.go`:
  - `Load(configPath string) (*Config, error)`:
    - If `configPath` is non-empty, load that file
    - Otherwise, search in order: `makefmt.yml`, `makefmt.yaml`, `.makefmt.yml`, `.makefmt.yaml`
    - If no file found, return `DefaultConfig()`
    - Unmarshal YAML into `Config`, then apply defaults for any missing fields
  - `Discover() string` — returns the first config file path found, or empty string
- [x] Create `internal/config/config_test.go`:
  - Test `DefaultConfig()` returns correct defaults
  - Test `Load()` with explicit path
  - Test `Load()` with discovery (create temp dir with config files)
  - Test `Load()` with no config file (returns defaults)
  - Test partial YAML (missing fields get defaults, not zero values)
  - Test invalid YAML (returns error)

### Files Created

- `internal/config/config.go`
- `internal/config/loader.go`
- `internal/config/config_test.go`

### Success Criteria

- `go test -race ./internal/config/...` passes
- Default config matches DESIGN.md schema defaults exactly
- YAML discovery follows the correct priority order
- Partial YAML files are merged with defaults (not zero-valued)

### Dependencies

None (defines types only; wired into CLI in Phase 5).

### External Dependency

Requires `gopkg.in/yaml.v3` (see Resolved Design Decisions).

---

## Phase 5 — CLI + Runner + Diff

Wire everything together: CLI flag parsing, the runner that orchestrates
parse → format → output, and unified diff generation.

### Tasks

- [x] Create `pkg/diff/diff.go`:
  - Hand-rolled Myers diff implementation (no external dependency)
  - `Unified(filename, old, new string) string` — generates a unified diff
  - Returns empty string if old == new
  - Standard unified diff format with `---`/`+++` headers and `@@` hunks
  - Context lines (default 3) around each change
- [x] Create `pkg/diff/diff_test.go`:
  - Test identical inputs → empty string
  - Test additions, deletions, modifications
  - Test output matches standard unified diff format
  - Test large files (performance sanity check)
  - Test empty inputs (both sides, one side)
- [x] Create `internal/runner/runner.go`:
  - `Options` struct: `Files []string`, `Check bool`, `Diff bool`, `Write bool`, `Stdin bool`, `ConfigPath string`, `Quiet bool`, `Verbose bool`
  - `Run(opts *Options) int` — main orchestration:
    1. Load config via `config.Load(opts.ConfigPath)`
    2. For each file (or stdin):
       - Read source
       - `parser.Parse(src)` → AST
       - `formatter.Run(nodes, cfg, rules)` → formatted AST
       - `writer.Write(formatted)` → output text
       - Mode dispatch: `--check` (compare, exit 1 if different), `--diff` (print unified diff), `--write` / default (write file or stdout)
    3. Return exit code (0, 1, or 2)
  - Handle stdin when no files given
  - Handle `--write` as default when files are given
  - Error handling: I/O errors → exit code 2
- [x] Create `internal/runner/runner_test.go`:
  - Test each mode (format to stdout, check, diff, write)
  - Test stdin handling
  - Test exit codes
  - Test error handling (missing file → exit 2)
- [x] Implement `cmd/makefmt/main.go`:
  - Use stdlib `flag` package (no cobra dependency)
  - Parse flags and translate to `runner.Options`
  - Flags: `-check`, `-diff`, `-w` (write), `-config`, `-q` (quiet), `-v` (verbose), `-version`
  - Note: stdlib `flag` uses single-dash (`-check`); support `--check` via `flag` default behavior
  - Call `runner.Run()` and `os.Exit()` with the result
  - Version vars (`version`, `commit`, `date`) set via ldflags at build time
  - Custom usage function for `--help` output matching DESIGN.md spec

### Files Created

- `pkg/diff/diff.go`
- `pkg/diff/diff_test.go`
- `internal/runner/runner.go`
- `internal/runner/runner_test.go`

### Files Modified

- `cmd/makefmt/main.go`
- `go.mod` / `go.sum`

### Success Criteria

- `make build` produces a working binary at `build/bin/makefmt`
- `echo "VAR:=val" | build/bin/makefmt` outputs `VAR := val`
- `build/bin/makefmt --check` exits 1 for unformatted input, 0 for formatted
- `build/bin/makefmt --diff` prints a unified diff
- `build/bin/makefmt --version` prints version
- `build/bin/makefmt Makefile` formats in-place (with `--write` default for file args)
- Exit code 2 for missing files or bad config
- `go test -race ./internal/runner/... ./pkg/diff/...` passes

### Dependencies

Phase 2 (writer), Phase 3 (formatter engine + rules), Phase 4 (config).

---

## Phase 6 — Advanced MVP Formatting Rules

Implement the remaining four MVP formatting rules. These are more complex than
the Phase 3 rules and benefit from having the full infrastructure in place.

### Tasks

- [x] Implement `internal/rules/format/backslash_align.go`:
  - Rule: `align_backslash_continuations`
  - Find continuation blocks (sequences of lines ending in `\`)
  - Align all `\` characters to a consistent column:
    - If `backslash_column > 0`: use that column
    - If `backslash_column == 0` (auto): use the longest line in the block + 1 space
  - Preserve content before the `\`; only adjust trailing whitespace before `\`
- [x] Implement `internal/rules/format/comment_spacing.go`:
  - Rule: `space_after_comment`
  - For `NodeComment` nodes with prefix `#` (single hash):
    - Ensure a space after `#` (e.g., `#comment` → `# comment`)
  - Skip (do not touch):
    - `##` comments (double hash)
    - `##@` section headers
    - `NodeBannerComment` nodes
    - Shebangs (`#!`)
    - Empty comments (`#` alone)
- [x] Implement `internal/rules/format/conditional_indent.go`:
  - Rule: `indent_conditionals`
  - For `NodeConditional` nodes: indent the body (Children) by `cfg.ConditionalIndent` spaces
  - `ifeq`/`ifdef`/`ifndef` open an indent level
  - `else` aligns with the opening directive (same indent as `ifeq`)
  - `endif` aligns with the opening directive
  - Handle nested conditionals (multiply indent level)
- [x] Implement `internal/rules/format/banner_preserve.go`:
  - Rule: `preserve_banner_comments`
  - This is a guard rule: ensure `NodeBannerComment` and `NodeSectionHeader` nodes pass through without modification from other rules
  - If prior rules inadvertently modified these nodes, restore from `Raw` field
  - Should run last in the rule chain
- [x] Create tests for each rule:
  - `internal/rules/format/backslash_align_test.go`
  - `internal/rules/format/comment_spacing_test.go`
  - `internal/rules/format/conditional_indent_test.go`
  - `internal/rules/format/banner_preserve_test.go`
- [x] Create golden file tests:
  - `testdata/backslash_align/input.mk` + `expected.mk`
  - `testdata/comment_spacing/input.mk` + `expected.mk`
  - `testdata/conditional_indent/input.mk` + `expected.mk`
  - `testdata/banner_preserve/input.mk` + `expected.mk`
- [x] Register all four rules in `internal/rules/registry.go` in the correct order
- [x] Create `testdata/full_format/input.mk` + `expected.mk`:
  - The comprehensive golden file test from DESIGN.md (lines 626-716)
  - Exercises all 8 rules together

### Files Created

- `internal/rules/format/backslash_align.go`
- `internal/rules/format/backslash_align_test.go`
- `internal/rules/format/comment_spacing.go`
- `internal/rules/format/comment_spacing_test.go`
- `internal/rules/format/conditional_indent.go`
- `internal/rules/format/conditional_indent_test.go`
- `internal/rules/format/banner_preserve.go`
- `internal/rules/format/banner_preserve_test.go`
- `testdata/backslash_align/input.mk`
- `testdata/backslash_align/expected.mk`
- `testdata/comment_spacing/input.mk`
- `testdata/comment_spacing/expected.mk`
- `testdata/conditional_indent/input.mk`
- `testdata/conditional_indent/expected.mk`
- `testdata/banner_preserve/input.mk`
- `testdata/banner_preserve/expected.mk`
- `testdata/full_format/input.mk`
- `testdata/full_format/expected.mk`

### Files Modified

- `internal/rules/registry.go`

### Success Criteria

- `go test -race ./internal/rules/format/...` passes
- Golden file tests pass for each individual rule
- `testdata/full_format` golden file test passes (all 8 rules applied together)
- Backslash alignment respects `backslash_column` config
- Comment spacing skips `##`, `##@`, banners, shebangs
- Conditional indentation handles nested conditionals
- Banner comments survive the full rule pipeline unmodified

### Dependencies

Phase 3 (formatter engine, registry, rule interface).

---

## Phase 7 — Polish, Integration Tests, Release Prep

Final quality pass before tagging v0.1.0.

### Tasks

- [x] Integration tests (`internal/runner/integration_test.go`):
  - Build the binary, then run it with `exec.CommandContext` against test fixtures
  - Test `--check` exit codes (0 for formatted, 1 for unformatted)
  - Test `--diff` output matches expected unified diff
  - Test `--write` actually modifies files
  - Test stdin/stdout piping
  - Test `--config` explicit path
  - Test `--version` output
  - Test exit code 2 for missing file
  - Test multiple files on command line
- [x] Fuzz test for parser (`internal/parser/fuzz_test.go`):
  - `func FuzzParse(f *testing.F)` — seed with 15 representative Makefile constructs, 165k+ executions with no panics
- [x] Dogfood: format the project's own `Makefile` with `makefmt` and verify no changes
  - Created `makefmt.yml` with `assignment_spacing: preserve` and `align_backslash_continuations: false`
- [x] Verify `make ci` passes (lint + test + build)
- [x] Verify `goreleaser check` passes with the fixed `.goreleaser.yml`
  - Fixed `archives.format` → `archives.formats` deprecation
- [x] Update `cmd/makefmt/main.go` help text to match DESIGN.md CLI spec
- [x] Create `makefmt.yml` dogfood config in repo root (with project defaults)
- [ ] Tag `v0.1.0` (deferred to user)

### Files Created

- `internal/runner/integration_test.go` (or `cmd/makefmt/main_test.go`)
- `internal/parser/fuzz_test.go`
- `makefmt.yml` (dogfood config)

### Success Criteria

- `make ci` passes
- `goreleaser check` passes
- Integration tests cover all CLI modes and exit codes
- Fuzz test runs without panics for a meaningful duration
- Project's own Makefile is formatted by makefmt
- `v0.1.0` tag created

### Dependencies

All prior phases (0-6).

---

## Phase 8 — Linter Engine + Rules (Post-MVP)

This phase is included for completeness. It is not part of the v0.1.0 release.

### Tasks

- [ ] Create `internal/linter/diagnostic.go`:
  - `Diagnostic` struct: `File`, `Line`, `Col`, `Severity`, `Rule`, `Message`
  - `Severity` enum: `Off`, `Warn`, `Error`
  - `Format() string` — produces `file:line:col: severity: message (rule-name)`
- [ ] Create `internal/linter/rule.go`:
  - `LintRule` interface: `Name() string`, `DefaultSeverity() Severity`, `Check(nodes []*Node, cfg *config.LintConfig) []Diagnostic`
- [ ] Create `internal/linter/engine.go`:
  - `Run(nodes []*Node, cfg *config.LintConfig, rules []LintRule) []Diagnostic`
  - Filters rules by configured severity (skip `Off`)
  - Collects and sorts diagnostics by file/line
- [ ] Implement initial lint rules:
  - `internal/rules/lint/recipe_tab.go` — recipe lines must start with tab
  - `internal/rules/lint/phony_declared.go` — targets without `.PHONY`
  - `internal/rules/lint/undefined_var.go` — `$(VAR)` with no visible assignment
- [ ] Register lint rules in `internal/rules/registry.go`
- [ ] Add `--lint` flag to CLI
- [ ] Wire lint mode into `runner.Run()`
- [ ] Add lint config section to `internal/config/config.go`

### Files Created

- `internal/linter/diagnostic.go`
- `internal/linter/rule.go`
- `internal/linter/engine.go`
- `internal/rules/lint/recipe_tab.go`
- `internal/rules/lint/phony_declared.go`
- `internal/rules/lint/undefined_var.go`

### Files Modified

- `internal/rules/registry.go`
- `cmd/makefmt/main.go`
- `internal/runner/runner.go`
- `internal/config/config.go`

### Success Criteria

- `makefmt --lint Makefile` reports diagnostics in the expected format
- Diagnostic output is compatible with Vim/Neovim errorformat
- Per-rule severity configuration works (error/warn/off)

### Dependencies

Phase 1 (parser/AST), Phase 4 (config types).

---

## MVP Rule Coverage Checklist

All 8 MVP formatting rules from DESIGN.md and their implementation phase:

| # | Rule                             | Config Key                         | Phase |
|---|----------------------------------|------------------------------------|-------|
| 1 | Trim trailing whitespace         | `trim_trailing_whitespace`         | 3     |
| 2 | Insert final newline             | `insert_final_newline`             | 3     |
| 3 | Max blank lines                  | `max_blank_lines`                  | 3     |
| 4 | Assignment spacing               | `assignment_spacing`               | 3     |
| 5 | Align backslash continuations    | `align_backslash_continuations`    | 6     |
| 6 | Space after comment              | `space_after_comment`              | 6     |
| 7 | Indent conditionals              | `indent_conditionals`              | 6     |
| 8 | Preserve banner comments         | `preserve_banner_comments`         | 6     |

---

## External Dependencies

| Dependency          | Package              | Purpose             | Phase |
|---------------------|----------------------|---------------------|-------|
| YAML parser         | `gopkg.in/yaml.v3`  | Config file loading | 4     |

The CLI uses stdlib `flag` (no external dependency). Diff generation is
hand-rolled in `pkg/diff/` (no external dependency).

---

## Resolved Design Decisions

All questions were resolved before implementation began.

### 1. CLI framework → stdlib `flag`

The CLI is a single command with ~8 flags and no subcommands. stdlib `flag`
keeps the dependency count low and is sufficient for the MVP. If subcommands
are needed post-MVP (`makefmt lint`, `makefmt fmt`), cobra can be added later.

### 2. YAML library → `gopkg.in/yaml.v3`

De facto standard Go YAML library. Well-tested, no transitive dependencies.
Performance is irrelevant for loading a small config file once at startup.

### 3. Diff → hand-rolled in `pkg/diff/`

Rather than depending on `github.com/pmezard/go-difflib` (unmaintained since
2013), we implement our own Myers diff in `pkg/diff/`. This is a public package
(not `internal/`) — the diff algorithm is general-purpose and not specific to
Makefile formatting, so there's no reason to lock it behind `internal/`. This
gives us full control over security updates, idiomatic Go style, and the output
format. The implementation is a standard Myers diff algorithm producing unified
diff output — well-documented in the literature and straightforward to
implement.

### 4. Node immutability → clone-on-write (default)

Each rule receives `[]*Node` and returns a new slice. Rules that modify a node
clone it first via `node.Clone()`. Rules that don't touch a node return the
same pointer. This balances correctness with performance. A future `--deep`
flag could enable full deep-clone of the entire AST before each rule for
debugging or strict correctness verification, but is not needed for MVP.

### 5. Continuation lines → store both forms

The `Raw` field preserves the original multi-line text (including `\` characters
and line breaks). Parsed fields (`VarValue`, `Targets`, etc.) contain the
joined/clean values. The writer uses `Raw` for untouched nodes and reconstructs
from fields for modified nodes. The backslash alignment rule specifically needs
the original line boundaries to re-align `\` characters.

### 6. Golden file test helpers → `internal/testutil/golden.go`

A shared helper that takes a `testing.T`, a testdata directory path, an optional
config override, and runs parse → format → write → compare. Keeps test files
focused on test cases, not infrastructure.

The helper supports a `-update` flag for regenerating golden files:

```bash
go test ./... -update     # regenerate all expected.mk from current output
```

When `-update` is passed, the helper writes actual formatter output to
`expected.mk` instead of comparing. This is the standard pattern in Go
formatting tools — every major formatter supports it:

**Reference implementations:**

- **gofumpt** (`github.com/mvdan/gofumpt`) — `testdata/` with txtar archives.
  Test helper in `format/format_test.go` uses `testscript` package. Supports
  `-update` to rewrite archive files.

- **yamlfmt** (`github.com/google/yamlfmt`) — `testdata/` with `before.yaml` /
  `after.yaml` pairs per feature directory. Test runner in
  `formatters/basic/basic_test.go` walks the testdata tree.

- **shfmt** (`github.com/mvdan/sh`) — `testdata/` with `.in` / `.out` file
  pairs. Test helper in `syntax/printer_test.go` uses `filepath.Walk` discovery.

Our approach is closest to yamlfmt/shfmt: `testdata/<rule>/input.mk` +
`expected.mk` pairs, with a shared helper in `internal/testutil/` that handles
the parse → format → write → compare cycle. The `-update` flag makes it easy to
bootstrap new golden files and update them after intentional formatting changes.

### 7. Banner preservation → guard rule (runs last)

`preserve_banner_comments` is a real formatting rule registered last in the
chain. It ensures that `NodeBannerComment` and `NodeSectionHeader` nodes pass
through unmodified. If a prior rule (e.g., `trim_trailing_whitespace`)
inadvertently modified these nodes, the guard restores them from the `Raw`
field. This is defensive and costs almost nothing.

### 8. Config defaults → `DefaultConfig()` constructor

`DefaultConfig()` returns a fully populated config with all defaults from
DESIGN.md. When loading YAML, we unmarshal into a `DefaultConfig()` instance
so that missing fields retain their defaults rather than becoming Go zero
values. This is the standard Go pattern for config with non-zero defaults.

### 9. Rule enable/disable → rules check their own config

Each rule checks its own config field and returns the input unchanged if
disabled. The engine always runs all rules in registered order. This keeps the
engine simple and puts the enable/disable logic where it belongs — in the rule
itself. The engine doesn't need to know which config keys map to which rules.

### 10. Version injection → ldflags via goreleaser

`goreleaser` sets `main.version`, `main.commit`, `main.date` at build time via
`-ldflags`. For local `make build`, the fallback reads from `git describe` or
defaults to `dev`. No `VERSION` file needed.
