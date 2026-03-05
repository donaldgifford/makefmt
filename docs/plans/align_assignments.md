# Plan: `align_assignments` Formatting Rule

## Summary

Implement the `align_assignments` formatting rule that column-aligns assignment
operators (`:=`, `?=`, `+=`, `=`, etc.) within groups of consecutive assignment
lines. This is already specified in DESIGN.md (lines 836-850) and the config
field exists (`AlignAssignments bool`, default `false`), but no rule
implementation exists yet.

**Goal**: Turn this:

```makefile
PROJECT_NAME := makefmt
PROJECT_OWNER := donaldgifford
DESCRIPTION := GNU Make formatter
PROJECT_URL := https://github.com/foo/bar
```

Into this:

```makefile
PROJECT_NAME  := makefmt
PROJECT_OWNER := donaldgifford
DESCRIPTION   := GNU Make formatter
PROJECT_URL   := https://github.com/foo/bar
```

## Design Decisions

### 1. Default value: change to `true`

The user wants aligned assignments as the default style. Change
`DefaultConfig()` from `AlignAssignments: false` to `AlignAssignments: true`.

### 2. Grouping rules (per DESIGN.md lines 838-843)

Consecutive `NodeAssignment` lines form a **group**. A group is broken by:

- Blank lines (`NodeBlankLine`)
- Comments (`NodeComment`, `NodeSectionHeader`, `NodeBannerComment`)
- Any non-assignment node (`NodeRule`, `NodeRecipe`, `NodeConditional`, etc.)

Each group is aligned independently. A single assignment on its own is a group
of one and gets no extra padding.

### 3. Alignment strategy: pad VarName with trailing spaces

Within a group, find the longest `VarName` length. For each assignment in the
group, right-pad `VarName` with spaces so all operators start in the same
column. Then clear `Raw` so the writer reconstructs from fields.

The writer (`writeAssignment` at `writer.go:98-106`) emits:
`VarName + " " + AssignOp + " " + VarValue`

So padding `VarName` to a uniform width is sufficient — the writer's single
space before the operator handles the rest.

Example with longest name being `PROJECT_OWNER` (13 chars):

```
VarName (padded)   writer adds   AssignOp   writer adds   VarValue
"PROJECT_NAME  "   " "           ":="       " "           "makefmt"
"PROJECT_OWNER "   " "           ":="       " "           "donaldgifford"
"DESCRIPTION   "   " "           ":="       " "           "GNU Make formatter"
```

### 4. Mixed operators within a group

Groups can contain mixed operators (`:=`, `?=`, `+=`, `=`). Operators have
different lengths (1-3 chars). Alignment is based on the **start column** of
the operator, not the `=` sign. This matches the convention in the project's
own Makefile:

```makefile
GO          ?= go
GO_PACKAGE  := github.com/foo/bar
GOOS        ?= $(shell $(GO) env GOOS)
```

The `?=` and `:=` start in the same column. The `=` signs don't line up, and
that's correct — aligning on operator start is the standard convention.

### 5. `override` prefix handling

The parser stores `override VAR` as the full `VarName` (e.g., `"override FOO"`).
The alignment should use the full VarName length including the `override ` prefix.
No special handling needed — it participates in grouping naturally.

### 6. Rule execution order

Must run **after** `AssignmentSpacing` (which normalizes single-space or
no-space around operators) and **before** `BackslashAlign`. The alignment rule
expects `Raw` to already be cleared by `AssignmentSpacing` in `space` mode, or
needs to clear it itself when `assignment_spacing` is `preserve`.

Registration in `register.go`:

```go
RegisterFormatRule(&format.AssignmentSpacing{})
RegisterFormatRule(&format.AlignAssignments{})  // <-- new
RegisterFormatRule(&format.BackslashAlign{})
```

### 7. Interaction with `assignment_spacing` modes

| `assignment_spacing` | `align_assignments` | Result |
|----------------------|---------------------|--------|
| `space`              | `true`              | Aligned with spaces: `VAR___:= val` |
| `space`              | `false`             | Single space: `VAR := val` |
| `no_space`           | `true`              | Aligned with no space after name: `VAR___:=val` |
| `no_space`           | `false`             | Compact: `VAR:=val` |
| `preserve`           | `true`              | Aligned (clears Raw, reconstructs with space) |
| `preserve`           | `false`             | Untouched |

For `no_space` + `align_assignments: true`: pad the VarName, then set `Raw`
directly as `paddedName + op + value` (no spaces around operator). The rule
needs to be aware of the spacing mode.

For `preserve` + `align_assignments: true`: the alignment rule takes precedence
and clears `Raw` to reconstruct. The operator will get spaces (writer default).

### 8. Continuation lines

Assignment nodes with backslash continuations have the continuation content
baked into `Raw` / `VarValue`. The alignment rule uses `VarName` length only
for column calculation, so continuations don't affect the logic. The `Raw`
field is cleared and the writer reconstructs from fields.

## Files to Change

### New files

| File | Purpose |
|------|---------|
| `internal/rules/format/align_assignments.go` | Rule implementation |
| `internal/rules/format/align_assignments_test.go` | Unit tests |
| `testdata/align_assignments/input.mk` | Golden file input |
| `testdata/align_assignments/expected.mk` | Golden file expected output |

### Modified files

| File | Change |
|------|--------|
| `internal/rules/register.go` | Register `AlignAssignments` after `AssignmentSpacing` |
| `internal/config/config.go` | Change default `AlignAssignments` from `false` to `true` |
| `testdata/assignment_spacing/expected.mk` | Update: assignments are now aligned by default |
| `testdata/full_format/expected.mk` | Update: assignments are now aligned by default |

## Implementation

### Step 1: Change default config

In `internal/config/config.go`, change line 46:

```go
AlignAssignments: true,
```

### Step 2: Implement the rule

Create `internal/rules/format/align_assignments.go`:

```go
package format

import (
    "strings"

    "github.com/donaldgifford/makefmt/internal/config"
    "github.com/donaldgifford/makefmt/internal/parser"
)

// AlignAssignments column-aligns assignment operators within groups
// of consecutive assignment lines.
type AlignAssignments struct{}

func (*AlignAssignments) Name() string {
    return "align_assignments"
}

func (*AlignAssignments) Format(
    nodes []*parser.Node,
    cfg *config.FormatterConfig,
) []*parser.Node {
    if !cfg.AlignAssignments {
        return nodes
    }

    result := make([]*parser.Node, len(nodes))
    copy(result, nodes)

    // Find groups of consecutive assignments and align each group.
    i := 0
    for i < len(result) {
        if result[i].Type != parser.NodeAssignment {
            i++
            continue
        }

        // Found start of a group — collect consecutive assignments.
        start := i
        for i < len(result) && result[i].Type == parser.NodeAssignment {
            i++
        }

        if i-start > 1 {
            alignGroup(result[start:i], cfg.AssignmentSpacing)
        }
    }

    return result
}

func alignGroup(group []*parser.Node, spacingMode string) {
    // Find the longest VarName in the group.
    maxLen := 0
    for _, n := range group {
        if l := len(n.Fields.VarName); l > maxLen {
            maxLen = l
        }
    }

    // Pad each VarName and clear Raw for reconstruction.
    for i, n := range group {
        clone := n.Clone()
        padded := clone.Fields.VarName +
            strings.Repeat(" ", maxLen-len(clone.Fields.VarName))

        switch spacingMode {
        case "no_space":
            raw := padded + clone.Fields.AssignOp
            if clone.Fields.VarValue != "" {
                raw += clone.Fields.VarValue
            }
            clone.Raw = raw
        default:
            // "space" or "preserve" — use writer reconstruction.
            clone.Fields.VarName = padded
            clone.Raw = ""
        }

        group[i] = clone
    }
}
```

### Step 3: Register the rule

In `internal/rules/register.go`, add after `AssignmentSpacing`:

```go
RegisterFormatRule(&format.AssignmentSpacing{})
RegisterFormatRule(&format.AlignAssignments{})  // new
```

### Step 4: Write unit tests

Create `internal/rules/format/align_assignments_test.go` with table-driven
tests covering:

1. **Basic alignment** — group of 3 assignments with different name lengths
2. **Single assignment** — no padding applied (group of 1)
3. **Multiple groups** — blank line separating two groups, each aligned independently
4. **Mixed operators** — `:=`, `?=`, `+=` in same group, aligned on operator start
5. **Disabled** — `AlignAssignments: false` returns nodes unchanged
6. **Already aligned** — idempotent, no changes needed
7. **`no_space` mode** — alignment works with `assignment_spacing: no_space`
8. **`override` prefix** — `override VAR` participates in alignment correctly
9. **Non-assignment breaks group** — comment between assignments creates two groups

### Step 5: Create golden file tests

`testdata/align_assignments/input.mk`:

```makefile
PROJECT_NAME := makefmt
PROJECT_OWNER := donaldgifford
DESCRIPTION := GNU Make formatter
PROJECT_URL := https://github.com/foo/bar

GO ?= go
GO_PACKAGE := github.com/foo/bar
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# This comment breaks the group
SINGLE_VAR := alone

A := 1
AB := 2
ABC := 3
```

`testdata/align_assignments/expected.mk`:

```makefile
PROJECT_NAME  := makefmt
PROJECT_OWNER := donaldgifford
DESCRIPTION   := GNU Make formatter
PROJECT_URL   := https://github.com/foo/bar

GO         ?= go
GO_PACKAGE := github.com/foo/bar
GOOS       ?= $(shell go env GOOS)
GOARCH     ?= $(shell go env GOARCH)

# This comment breaks the group
SINGLE_VAR := alone

A   := 1
AB  := 2
ABC := 3
```

### Step 6: Update existing golden files

Since the default changes to `align_assignments: true`, existing golden file
expected outputs need to be updated where they contain consecutive assignments.
Run `make test` to identify failures, then run with `-update` flag to
regenerate expected files, and verify the changes look correct.

### Step 7: Update documentation

Update `docs/RULES.md` to document the `align_assignments` rule behavior and
configuration. Update `docs/USAGE.md` if the config example shows the old
default.

## Testing Strategy

1. **Unit tests** — `align_assignments_test.go` with table-driven tests
2. **Golden file test** — `testdata/align_assignments/` directory
3. **Existing golden files** — verify they pass after updating expected output
4. **Integration** — `make test` runs all tests with race detector
5. **Lint** — `make lint` passes
6. **Manual** — run `makefmt` on the project's own Makefile and verify output

## Risks and Edge Cases

- **Continuation lines**: Assignments with `\` continuations store multi-line
  content in VarValue. Alignment only pads VarName, so this should work, but
  needs testing to confirm the writer handles it correctly when Raw is cleared.
- **Very long variable names**: One long name in a group could create excessive
  padding. This is acceptable — it matches the behavior of every other
  alignment formatter (gofmt struct tags, clang-format, etc.).
- **Tabs in VarName**: The parser trims whitespace, so VarName should never
  contain tabs. No special handling needed.
- **Golden file churn**: Changing the default to `true` affects existing golden
  files. This is a one-time migration.
