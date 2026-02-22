# Implementation Plan: `align_assignments`

> **Feature branch**: `feat/align-assignments`
> **Plan document**: `docs/plans/align_assignments.md`
> **Design reference**: `docs/DESIGN.md` lines 836-850, 471-472, 983-986

---

## Phase 1: Core Rule Implementation

Implement the `AlignAssignments` format rule with no changes to defaults or
existing tests. The rule is registered but disabled by default, so existing
behavior is unchanged.

### Tasks

- [x] **1.1** Create `internal/rules/format/align_assignments.go`
  - Implement the `AlignAssignments` struct with `Name()` returning
    `"align_assignments"`
  - Implement `Format()` that:
    - Returns early if `cfg.AlignAssignments` is `false`
    - Copies the node slice
    - Iterates to find groups of consecutive `NodeAssignment` nodes
    - For groups of size > 1, calls `alignGroup()`
  - Implement `alignGroup(group []*parser.Node, spacingMode string)` that:
    - Finds the longest `VarName` in the group
    - Clones each node, right-pads `VarName` with spaces to match the
      longest name
    - For `spacingMode == "no_space"`: sets `Raw` directly to
      `paddedName + op [+ value]` (no spaces around operator)
    - For all other modes (`"space"`, `"preserve"`): sets
      `clone.Fields.VarName` to the padded value and clears `Raw` so the
      writer reconstructs with spaces
  - The rule must strip existing over-padding: always pad to exactly the
    longest VarName length in the group, normalizing any wider padding down.
    The algorithm uses `strings.TrimRight(VarName, " ")` before measuring
    lengths to ensure pre-existing padding doesn't inflate the column width.

- [x] **1.2** Register the rule in `internal/rules/register.go`
  - Add `RegisterFormatRule(&format.AlignAssignments{})` immediately after
    the `AssignmentSpacing` registration (after line 13, before line 15)
  - This positions it after operator spacing normalization and before
    backslash alignment

- [x] **1.3** Verify existing tests still pass
  - Run `make test` — all tests must pass since `AlignAssignments` defaults
    to `false` and the rule returns early

### Success Criteria

- `make test` passes with 0 failures
- `make lint` passes with 0 issues
- The new rule file compiles and is registered in the pipeline
- When `AlignAssignments` is `false` (the current default), formatter output
  is identical to before

---

## Phase 2: Unit Tests

Write table-driven unit tests for the rule covering all documented behavior
and edge cases.

### Tasks

- [ ] **2.1** Create `internal/rules/format/align_assignments_test.go` with
  the following test cases:

  **Basic behavior:**
  - [ ] **Group of 3 assignments** — different VarName lengths, verify
    operators align to the longest name's column
  - [ ] **Single assignment** — group of 1, verify no padding is added
  - [ ] **Already aligned** — idempotent, output matches input

  **Grouping:**
  - [ ] **Blank line breaks group** — two groups separated by
    `NodeBlankLine`, each aligned independently
  - [ ] **Comment breaks group** — `NodeComment` between assignments creates
    two separate groups
  - [ ] **Non-assignment breaks group** — `NodeRule` or `NodeConditional`
    between assignments creates separate groups

  **Operator handling:**
  - [ ] **Mixed operators** — `:=`, `?=`, `+=` in the same group, all
    operator starts align to the same column

  **Config interactions:**
  - [ ] **Disabled** — `AlignAssignments: false` returns nodes unmodified
    (same slice)
  - [ ] **`no_space` mode** — `AssignmentSpacing: "no_space"` produces
    aligned output without spaces around operators (verify via `Raw` field)
  - [ ] **`preserve` mode** — `AssignmentSpacing: "preserve"` with
    `AlignAssignments: true` clears `Raw` and reconstructs with spaces

  **Edge cases:**
  - [ ] **`override` prefix** — `override FOO` participates in alignment
    using full VarName length (including `override ` prefix)
  - [ ] **Empty VarValue** — bare `VAR :=` (no value) aligns correctly,
    no trailing whitespace introduced
  - [ ] **Over-padded input** — assignments with excessive existing padding
    are normalized down to the minimum alignment column
  - [ ] **Continuation line in group** — an assignment whose VarValue is
    space-joined continuation content (e.g., `main.go utils.go handler.go`)
    participates in alignment correctly; VarName length determines column,
    VarValue content is irrelevant to alignment

- [ ] **2.2** Verify writer output for each test case
  - Each test should call `formatter.Write()` on the result and verify the
    serialized text matches expectations (same pattern as
    `assignment_spacing_test.go`)

### Success Criteria

- `make test` passes, including all new test cases
- `make lint` passes
- Tests cover: basic alignment, grouping boundaries, mixed operators,
  all 3 spacing modes, disabled mode, override prefix, empty values,
  over-padding, continuation lines

---

## Phase 3: Default Change, Golden Files, and Test Updates

Change the default value of `AlignAssignments` to `true`, add new golden
file tests, and update all affected existing tests and golden files in a
single phase. These changes are tightly coupled — the golden tests need the
default enabled, and changing the default requires updating existing goldens.

### Tasks

- [ ] **3.1** Change default in `internal/config/config.go`
  - Set `AlignAssignments: true` in `DefaultConfig()` (line 46)

- [ ] **3.2** Update `internal/config/config_test.go`
  - Change `TestDefaultConfig` expectation at line 24 from `false` to `true`

- [ ] **3.3** Create `testdata/align_assignments/input.mk` with:
  - A group of 4 assignments with varying name lengths (e.g.,
    `PROJECT_NAME`, `PROJECT_OWNER`, `DESCRIPTION`, `PROJECT_URL`)
  - A blank line, then a second group with mixed operators (`:=`, `?=`)
  - A comment that breaks the group
  - A single isolated assignment (group of 1)
  - A final group of short names to test small padding (e.g., `A`, `AB`,
    `ABC`)
  - An over-padded group that should be normalized down

- [ ] **3.4** Create `testdata/align_assignments/expected.mk` with the
  correctly aligned output

- [ ] **3.5** Update golden file: `testdata/assignment_spacing/expected.mk`
  - Current file has 6 consecutive assignments (`PROJECT_NAME`,
    `PROJECT_OWNER`, `DESCRIPTION`, `GO`, `GO_PACKAGE`, `VERSION`) that
    will now be aligned
  - Run `go test ./internal/rules/format/ -run TestGoldenFiles/assignment_spacing -update`
    to regenerate, then manually verify the output is correct

- [ ] **3.6** Update golden file: `testdata/full_format/expected.mk`
  - Has two groups of consecutive assignments separated by section headers:
    - Group 1 (lines 3-5): `PROJECT_NAME`, `PROJECT_OWNER`, `DESCRIPTION`
    - Group 2 (lines 9-10): `GO`, `GO_PACKAGE`
  - Run with `-update` and verify

- [ ] **3.7** Verify remaining golden files are unaffected
  - `testdata/conditional_indent/` — assignments are separated by
    `else` (NodeConditional), so they form groups of 1 each. **No change
    expected.**
  - `testdata/backslash_align/` — single assignment (group of 1). **No
    change expected.**
  - `testdata/blank_lines/` — single assignment. **No change expected.**
  - `testdata/trailing_whitespace/` — assignments separated by blank line.
    **No change expected.**
  - `testdata/banner_preserve/` — no assignments. **No change expected.**
  - `testdata/comment_spacing/` — no assignments. **No change expected.**
  - Run `make test` to confirm all golden tests pass

- [ ] **3.8** Verify integration tests are unaffected
  - All integration tests use single-assignment inputs (groups of 1),
    so alignment produces no changes. Run `make test` to confirm.

- [ ] **3.9** Update `makefmt.yml` — add `align_assignments: true` explicitly
  ```yaml
  formatter:
    assignment_spacing: preserve
    align_assignments: true
    align_backslash_continuations: false
  ```

### Success Criteria

- `make test` passes with 0 failures
- `make lint` passes with 0 issues
- `DefaultConfig().Formatter.AlignAssignments == true`
- All golden files produce correctly aligned output
- New golden test exercises multiple groups, mixed operators, single
  assignments, over-padding normalization
- Integration tests pass unchanged
- `makefmt.yml` explicitly sets `align_assignments: true`

---

## Phase 4: Documentation

Update user-facing documentation to reflect the new rule and default.

### Tasks

- [ ] **4.1** Update `docs/RULES.md`
  - Add a new section between `assignment_spacing` (rule 4) and
    `align_backslash_continuations` (currently rule 5) — renumber subsequent
    rules (5 becomes 6, 6 becomes 7, etc.)
  - Document:
    - Config key: `align_assignments`
    - Type: `bool`
    - Default: `true`
    - Behavior: aligns assignment operators within consecutive groups
    - Grouping rules: blank lines, comments, and non-assignment lines
      break groups
    - Over-padding: existing excessive padding is normalized down
    - Before/after example
  - Update the "Rule Execution Order" section at the bottom to include
    `align_assignments` between `assignment_spacing` and
    `align_backslash_continuations`

- [ ] **4.2** Update `docs/USAGE.md`
  - Change the config reference at line 92 from
    `align_assignments: false` and `# Default: false (reserved for future use)`
    to `align_assignments: true` with updated comment
  - Add an `#### align_assignments` subsection in the Configuration Keys
    section (after `trim_trailing_whitespace`, before `assignment_spacing`)
    explaining the behavior

- [ ] **4.3** Update `README.md` if it references the config defaults

### Success Criteria

- Documentation accurately describes the new rule, its default, and
  its configuration
- The rule execution order list is correct and complete
- Config reference shows the correct default value

---

## Phase 5: Final Validation

End-to-end verification that everything works together.

### Tasks

- [ ] **5.1** Run `make ci` (lint + test + build)
- [ ] **5.2** Run `makefmt` on the project's own `Makefile` and verify
  the output preserves the existing aligned style (since the Makefile is
  already aligned, the formatter should be idempotent)
- [ ] **5.3** Run `makefmt --diff Makefile` and verify zero diff
  - The project's `makefmt.yml` sets `assignment_spacing: preserve` and
    `align_assignments: true`. Since `preserve` mode in `AssignmentSpacing`
    returns nodes unmodified (Raw intact), `AlignAssignments` will clone,
    pad VarName, and clear Raw. The writer reconstructs as
    `PADDED_VAR := val` (space style). Since the existing Makefile already
    uses space style with correct alignment, the output should match.
  - If there IS a diff, investigate whether `preserve` mode needs special
    handling — see Resolved Question 2 below for the investigation plan.
- [ ] **5.4** Run `makefmt --check Makefile` and verify exit code 0
- [ ] **5.5** Test with `align_assignments: false` in config to verify
  disabling works
- [ ] **5.6** Test with `assignment_spacing: no_space` + `align_assignments: true`
  to verify the no_space alignment path works end-to-end

### Success Criteria

- `make ci` passes
- `makefmt` is idempotent on its own Makefile (zero diff)
- `makefmt --check Makefile` exits 0
- Disabling via config works correctly
- `no_space` + alignment works correctly

---

## Resolved Questions

### 1. Golden test config override — RESOLVED: Combine phases

Phases 3 and 4 from the original plan are combined into a single Phase 3.
The golden test and default change are tightly coupled. Doing them together
avoids temporary workarounds and is the simplest approach.

### 3. Over-padding normalization — RESOLVED: Yes, strip it

The rule normalizes over-padded assignments down to the minimum column needed.
This is standard opinionated formatter behavior (same as `gofmt` normalizing
struct tag alignment). The algorithm trims existing trailing spaces from
VarName before measuring lengths, ensuring pre-existing padding doesn't
inflate the column width. This guarantees idempotent output.

### 4. Continuation line interaction — RESOLVED: Test it

Added as a required test case in Phase 2 (task 2.1). The existing behavior
where `assignment_spacing: space` collapses continuation lines into a single
line is preserved — `align_assignments` doesn't change this. A test case
verifies that a continuation-line assignment in a group aligns correctly
based on VarName length.

### 5. Project `makefmt.yml` update — RESOLVED: Yes, add explicitly

`align_assignments: true` will be added to `makefmt.yml` in Phase 3 (task
3.9). This makes the config self-documenting and prepares for a future
`makefmt init` command that generates explicit config files.

---

## Outstanding Investigation: `preserve` + `align_assignments` interaction

The project's `makefmt.yml` uses `assignment_spacing: preserve`. When
combined with `align_assignments: true`, the following happens:

1. `AssignmentSpacing` rule sees `preserve` mode and returns nodes
   **unmodified** (same pointers, `Raw` still set to original text)
2. `AlignAssignments` rule clones nodes, pads `VarName`, clears `Raw`
3. Writer reconstructs as `PADDED_VAR := val` (always uses spaces —
   this is the writer's hardcoded format)

This means `preserve` + `align_assignments: true` effectively becomes
`space` + `align_assignments: true`. For the project's own Makefile this
is fine (it already uses space style). But for a user who has
`assignment_spacing: preserve` because they use `no_space` style, enabling
alignment would silently switch their spacing to `space` style.

**Investigation needed during Phase 5 (task 5.3):**

If `makefmt --diff Makefile` shows zero diff, the current approach works
for the project. But the general case of `preserve` + alignment may need
one of these fixes:

**(a)** Accept the behavior — alignment implies spacing normalization.
Document that `align_assignments: true` uses `space` style unless
`assignment_spacing: no_space` is explicitly set.

**(b)** Have the alignment rule inspect the original `Raw` to detect the
spacing style and preserve it during reconstruction. This adds complexity
but respects the `preserve` intent.

**(c)** Treat `preserve` + `align_assignments: true` as a configuration
conflict and document that users should set an explicit `assignment_spacing`
when enabling alignment.

This will be resolved during Phase 5 based on testing results.
