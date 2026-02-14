# makefmt(1) — CLI Reference

## NAME

`makefmt` — format GNU Makefiles

## SYNOPSIS

```
makefmt [flags] [files...]
```

## DESCRIPTION

`makefmt` formats GNU Makefiles by parsing them into an AST, applying a
pipeline of formatting rules, and writing the result. It is designed to
produce consistent, readable Makefiles with zero configuration needed.

When no files are given, `makefmt` reads from stdin and writes the
formatted result to stdout.

When files are given, `makefmt` reads each file and writes the formatted
result to stdout by default. Use `-w` to write changes back to the file
in-place.

## FLAGS

| Flag | Description |
|------|-------------|
| `--check` | Exit with code 1 if any file is not already formatted. Does not produce output. |
| `--diff` | Print a unified diff of the changes that would be made. |
| `-w` | Write the formatted result back to the source file(s) in-place. |
| `--config <path>` | Path to a config file. Overrides automatic config discovery. |
| `-q` | Quiet mode. Suppress informational output. |
| `-v` | Verbose mode. Print file names as they are processed. |
| `--version` | Print version information and exit. |

Flags can be combined. For example, `--check --diff` prints a diff and
exits with code 1 if any file needs formatting.

## EXIT CODES

| Code | Meaning |
|------|---------|
| 0 | Success. All files are formatted (or formatting completed without error). |
| 1 | Formatting needed. At least one file is not formatted (with `--check`). |
| 2 | Usage or I/O error. Invalid flags, missing files, or read/write failures. |

## CONFIGURATION

### Discovery order

When `--config` is not specified, `makefmt` searches the current working
directory for a config file. The first match wins:

1. `makefmt.yml`
2. `makefmt.yaml`
3. `.makefmt.yml`
4. `.makefmt.yaml`

If no config file is found, built-in defaults are used. Partial config
files are supported — any fields not specified retain their default values.

### Full configuration reference

```yaml
formatter:
  # Indentation character for recipes.
  # Options: "tab"
  # Default: "tab"
  indent_style: tab

  # Tab display width (used for alignment calculations).
  # Default: 4
  tab_width: 4

  # Maximum consecutive blank lines allowed. Extra blank lines are collapsed.
  # Set to -1 to disable (preserve all blank lines).
  # Default: 2
  max_blank_lines: 2

  # Ensure the file ends with exactly one newline.
  # Default: true
  insert_final_newline: true

  # Remove trailing spaces and tabs from every line.
  # Default: true
  trim_trailing_whitespace: true

  # Align assignment operators in consecutive assignment blocks.
  # Default: false (reserved for future use)
  align_assignments: false

  # Spacing around assignment operators (:=, ?=, +=, =).
  # Options: "space" (VAR := val), "no_space" (VAR:=val), "preserve"
  # Default: "space"
  assignment_spacing: space

  # Sort prerequisites alphabetically in rule declarations.
  # Default: false (reserved for future use)
  sort_prerequisites: false

  # Align trailing backslashes in continuation blocks to a consistent column.
  # Default: true
  align_backslash_continuations: true

  # Target column for backslash alignment (1-indexed). Set to 0 for auto
  # mode (aligns to longest content line + 1 space).
  # Default: 79
  backslash_column: 79

  # Ensure a space after # in single-hash comments.
  # Skips: ##, ##@, shebangs (#!), empty comments (#), banner comments.
  # Default: true
  space_after_comment: true

  # Indent the body of conditional blocks (ifeq/ifdef/ifndef/else/endif).
  # Default: true
  indent_conditionals: true

  # Number of spaces for conditional indentation.
  # Default: 2
  conditional_indent: 2

  # Recipe line prefix handling.
  # Options: "preserve"
  # Default: "preserve"
  recipe_prefix: preserve

lint:
  # Lint rule severity overrides (post-MVP).
  # Map of rule name to severity: "off", "warn", "error".
  rules: {}

  # File patterns to exclude from linting.
  exclude: []
```

### Configuration keys

#### `indent_style`

Indentation character used in recipe lines. Currently only `"tab"` is
supported, matching GNU Make's requirement.

#### `tab_width`

Display width of a tab character. Used for alignment calculations (e.g.,
backslash alignment). Does not change the indentation character.

#### `max_blank_lines`

Maximum number of consecutive blank lines allowed. Runs of blank lines
exceeding this limit are collapsed. Set to `-1` to disable blank line
collapsing.

#### `insert_final_newline`

When `true`, ensures the file ends with exactly one newline. Trailing
blank lines are removed and the writer appends a single newline.

#### `trim_trailing_whitespace`

When `true`, removes trailing spaces and tabs from every line in the
file, including recipe lines, comments, and raw blocks.

#### `assignment_spacing`

Controls whitespace around assignment operators (`:=`, `?=`, `+=`, `=`).

- `"space"` — `VAR := value` (one space on each side)
- `"no_space"` — `VAR:=value` (no spaces)
- `"preserve"` — leave existing spacing unchanged

#### `align_backslash_continuations`

When `true`, aligns trailing backslashes in continuation blocks to a
consistent column, determined by `backslash_column`.

#### `backslash_column`

The target column (1-indexed) for backslash alignment. Set to `0` for
auto mode, which aligns to the longest content line plus one space.

#### `space_after_comment`

When `true`, ensures a space follows `#` in single-hash comments
(`#comment` becomes `# comment`). Does not modify `##`, `##@`, shebangs
(`#!`), empty comments (`#`), or banner comments.

#### `indent_conditionals`

When `true`, indents the body of conditional blocks (`ifeq`, `ifdef`,
`ifndef`). The opening directive, `else`, and `endif` are not indented
relative to each other — only the body lines between them are indented.

#### `conditional_indent`

Number of spaces used per nesting level when `indent_conditionals` is
enabled. Nested conditionals increase the indent level.

#### `recipe_prefix`

Controls recipe line prefix handling. Currently only `"preserve"` is
supported, which leaves recipe line prefixes unchanged.

## EXAMPLES

Format a single file to stdout:

```bash
makefmt Makefile
```

Format multiple files in-place:

```bash
makefmt -w Makefile Makefile.inc
```

Check formatting in CI (exits 1 if changes needed):

```bash
makefmt --check Makefile
```

Preview changes as a unified diff:

```bash
makefmt --diff Makefile
```

Check and show diff together:

```bash
makefmt --check --diff Makefile
```

Format from stdin (pipe):

```bash
cat Makefile | makefmt
```

Use a specific config file:

```bash
makefmt --config path/to/makefmt.yml Makefile
```

Verbose mode (print filenames as processed):

```bash
makefmt -v -w Makefile *.mk
```

Print version:

```bash
makefmt --version
```

## ENVIRONMENT

No environment variables are currently used by `makefmt`. All
configuration is done via config files and command-line flags.

## FILES

- `makefmt.yml` — primary config file name
- `makefmt.yaml` — alternate config file name
- `.makefmt.yml` — hidden config file name
- `.makefmt.yaml` — hidden config file name (alternate)

## SEE ALSO

- [Formatting Rules Reference](RULES.md) — detailed documentation for each rule
- [Design Document](DESIGN.md) — architecture and design decisions
- [gofmt](https://pkg.go.dev/cmd/gofmt) — Go formatter (UX inspiration)
- [yamlfmt](https://github.com/google/yamlfmt) — YAML formatter (UX inspiration)
