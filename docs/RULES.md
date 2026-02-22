# Formatting Rules Reference

## Overview

makefmt applies formatting rules as a pipeline. Each rule receives the full
AST (a list of nodes), applies its transformation, and returns a new AST.
Rules are executed in a fixed order — the output of one rule becomes the
input of the next.

All rules implement the `FormatRule` interface:

```go
type FormatRule interface {
    Name() string
    Format(nodes []*parser.Node, cfg *config.FormatterConfig) []*parser.Node
}
```

Rules must not mutate their input nodes. They return new or cloned nodes
when changes are needed, and pass through unmodified nodes otherwise.

## Formatting Rules

### 1. `trim_trailing_whitespace`

Removes trailing spaces and tabs from every line.

| | |
|---|---|
| **Config key** | `trim_trailing_whitespace` |
| **Type** | `bool` |
| **Default** | `true` |

Applies to all node types including recipe lines, comments, assignments,
and raw blocks. Multi-line raw fields (continuation blocks) have each
line trimmed individually.

**Before:**

```makefile
CC := gcc····
CFLAGS := -Wall··
```

(where `·` represents trailing spaces)

**After:**

```makefile
CC := gcc
CFLAGS := -Wall
```

### 2. `insert_final_newline`

Ensures the file ends with exactly one newline.

| | |
|---|---|
| **Config key** | `insert_final_newline` |
| **Type** | `bool` |
| **Default** | `true` |

Trailing blank lines at the end of the file are removed. The writer
always appends a single newline after the last node, producing exactly
one trailing newline.

**Before:**

```makefile
all: build
	@echo "done"



```

(three trailing blank lines)

**After:**

```makefile
all: build
	@echo "done"
```

(exactly one trailing newline)

### 3. `max_blank_lines`

Collapses consecutive blank lines to a configurable maximum.

| | |
|---|---|
| **Config key** | `max_blank_lines` |
| **Type** | `int` |
| **Default** | `2` |
| **Disable** | Set to `-1` to preserve all blank lines |

Runs of blank lines exceeding the configured maximum are collapsed. This
keeps logical sections visually separated without excessive vertical
whitespace.

**Before** (with `max_blank_lines: 2`):

```makefile
build:
	@go build ./...




test:
	@go test ./...
```

(four blank lines between targets)

**After:**

```makefile
build:
	@go build ./...


test:
	@go test ./...
```

(collapsed to two blank lines)

### 4. `assignment_spacing`

Normalizes whitespace around assignment operators (`:=`, `?=`, `+=`, `=`).

| | |
|---|---|
| **Config key** | `assignment_spacing` |
| **Type** | `string` |
| **Default** | `"space"` |
| **Options** | `"space"`, `"no_space"`, `"preserve"` |

- `"space"` — ensures one space on each side of the operator
- `"no_space"` — removes all spaces around the operator
- `"preserve"` — leaves existing spacing unchanged

**Before** (with `assignment_spacing: space`):

```makefile
PROJECT_NAME:=my-project
PROJECT_OWNER :=donaldgifford
DESCRIPTION:= A project
GO_PACKAGE:=github.com/foo/bar
VERSION+=extra
```

**After:**

```makefile
PROJECT_NAME := my-project
PROJECT_OWNER := donaldgifford
DESCRIPTION := A project
GO_PACKAGE := github.com/foo/bar
VERSION += extra
```

### 5. `align_assignments`

Column-aligns assignment operators within groups of consecutive assignment lines.

| | |
|---|---|
| **Config key** | `align_assignments` |
| **Type** | `bool` |
| **Default** | `true` |

**Grouping**: Consecutive assignment lines form a group. A group is broken by a
blank line, a comment, or any non-assignment line (rule, conditional, etc.).
Each group is aligned independently. A single assignment is a group of one and
receives no padding.

**Operator alignment**: All operator starts align to the column after the longest
variable name in the group. Mixed operators (`:=`, `?=`, `+=`) in the same group
all start at the same column.

**Over-padding**: If a group has been padded wider than necessary (e.g. by prior
manual editing or a previous run with different variables), the rule normalizes it
down to the minimum required column. Output is always idempotent.

**Before:**

```makefile
PROJECT_NAME := makefmt
PROJECT_OWNER := donaldgifford
DESCRIPTION := GNU Make formatter
```

**After:**

```makefile
PROJECT_NAME  := makefmt
PROJECT_OWNER := donaldgifford
DESCRIPTION   := GNU Make formatter
```

### 6. `align_backslash_continuations`

Aligns trailing backslashes in continuation blocks to a consistent column.

| | |
|---|---|
| **Config key** | `align_backslash_continuations` |
| **Type** | `bool` |
| **Default** | `true` |

Related setting:

| | |
|---|---|
| **Config key** | `backslash_column` |
| **Type** | `int` |
| **Default** | `79` |
| **Auto mode** | Set to `0` — aligns to the longest content line + 1 space |

Each continuation line is padded so its trailing backslash sits at the
target column. If a content line is longer than the target column, at
least one space is preserved before the backslash.

**Before:**

```makefile
release:
	@if [ -z "$(TAG)" ]; then \
		echo "Error: TAG is required"; \
			exit 1; \
	fi
```

(backslashes at inconsistent columns)

**After** (with `backslash_column: 79`):

```makefile
release:
	@if [ -z "$(TAG)" ]; then                                                    \
		echo "Error: TAG is required";                                              \
			exit 1;                                                                    \
	fi
```

(backslashes aligned to column 79)

### 7. `space_after_comment`

Ensures a space after `#` in single-hash comments.

| | |
|---|---|
| **Config key** | `space_after_comment` |
| **Type** | `bool` |
| **Default** | `true` |

**Skips the following comment types (leaves them unchanged):**

- `##` — double-hash comments (often used for help text)
- `##@` — section headers
- `#!` — shebangs
- `#` — empty comments (just `#` with nothing after)
- Banner comments (decorative separators like `###############`)

**Before:**

```makefile
#comment without space
# comment with space
## double hash
##@ Section Header
#
#!/bin/bash
#another
```

**After:**

```makefile
# comment without space
# comment with space
## double hash
##@ Section Header
#
#!/bin/bash
# another
```

Note: `##`, `##@`, `#`, and `#!` lines are unchanged. Only single-hash
comments with content have spacing normalized.

### 8. `indent_conditionals`

Indents the body of conditional blocks (`ifeq`, `ifneq`, `ifdef`, `ifndef`).

| | |
|---|---|
| **Config key** | `indent_conditionals` |
| **Type** | `bool` |
| **Default** | `true` |

Related setting:

| | |
|---|---|
| **Config key** | `conditional_indent` |
| **Type** | `int` |
| **Default** | `2` |

The opening directive (`ifeq`, `ifdef`, etc.), `else`, and `endif` are
kept at the current nesting level. Only the body lines between them are
indented. Nested conditionals increase the indent level.

**Before:**

```makefile
ifdef DEBUG
CFLAGS := -g
else
CFLAGS := -O2
endif
```

**After** (with `conditional_indent: 2`):

```makefile
ifdef DEBUG
  CFLAGS := -g
else
  CFLAGS := -O2
endif
```

### 9. `preserve_banner_comments`

Ensures banner comments and section headers pass through unmodified.

| | |
|---|---|
| **Config key** | `preserve_banner_comments` |
| **Type** | Always active (no config toggle) |

This is a guard rule that runs last in the pipeline. It ensures that
banner comments (decorative separators like `###############`) and
section headers (`##@ Development`) are never accidentally modified by
prior rules.

Banner comments and section headers have their `Raw` field set by the
parser, so they pass through the writer verbatim. This rule documents
and enforces that invariant.

**Example (preserved as-is):**

```makefile
###############
##@ Development

# regular comment

########
##@ Help
```

## Rule Execution Order

Rules are applied in the following fixed order:

1. `trim_trailing_whitespace` — clean up whitespace first
2. `insert_final_newline` — normalize file ending
3. `max_blank_lines` — collapse excessive blank lines
4. `assignment_spacing` — normalize assignment operators
5. `align_assignments` — column-align operators within groups
6. `align_backslash_continuations` — align continuation backslashes
7. `space_after_comment` — normalize comment spacing
8. `indent_conditionals` — indent conditional bodies
9. `preserve_banner_comments` — guard rule (runs last)

This order matters. For example, trailing whitespace is trimmed before
backslash alignment, so the aligner works with clean lines. Assignment
spacing normalizes operators before conditional indentation adds
prefixes.

## Lint Rules (Planned)

Lint rules are planned for a future release. Unlike formatting rules,
lint rules inspect the AST and report diagnostics without modifying the
code. Each diagnostic has a severity (`off`, `warn`, `error`).

Planned lint rules include:

- **`missing_phony`** — warn when targets with no output file are not
  declared `.PHONY`
- **`recipe_shell_safety`** — warn about unsafe shell patterns in
  recipe lines (unquoted variables, missing `set -e`)
- **`unused_variable`** — warn about variables that are assigned but
  never referenced
- **`duplicate_target`** — warn when the same target appears in
  multiple rules

Lint rules will be configurable via the `lint.rules` section of the
config file, with per-rule severity overrides.
