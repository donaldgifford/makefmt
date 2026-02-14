# makefmt

A fast, opinionated GNU Make formatter written in Go. Single binary, zero
dependencies. Designed for format-on-save (conform.nvim) and CI checks.

Follows the UX patterns of `gofmt`, `yamlfmt`, and `terraform fmt`.

## Features

- Formats GNU Makefiles with sensible defaults
- Single static binary — no runtime dependencies
- Reads from stdin or files, writes to stdout or in-place
- `--check` mode for CI (exit 1 if unformatted)
- `--diff` mode for previewing changes
- Configurable via `makefmt.yml` with automatic discovery
- 8 built-in formatting rules (whitespace, spacing, alignment, indentation)
- Preserves banner comments and section headers (`##@`)

## Installation

### Go install

```bash
go install github.com/donaldgifford/makefmt/cmd/makefmt@latest
```

### Build from source

```bash
git clone https://github.com/donaldgifford/makefmt.git
cd makefmt
make build
# Binary is at build/bin/makefmt
```

## Quick Start

Format a Makefile and print to stdout:

```bash
makefmt Makefile
```

Format in-place:

```bash
makefmt -w Makefile
```

Check if files are formatted (for CI):

```bash
makefmt --check Makefile
```

Show a diff of what would change:

```bash
makefmt --diff Makefile
```

Read from stdin:

```bash
cat Makefile | makefmt
```

## Configuration

makefmt looks for a config file in the current directory. First match wins:

1. `makefmt.yml`
2. `makefmt.yaml`
3. `.makefmt.yml`
4. `.makefmt.yaml`

Or specify one explicitly with `--config path/to/config.yml`.

Minimal example:

```yaml
formatter:
  max_blank_lines: 1
  assignment_spacing: no_space
  indent_conditionals: false
```

Any fields not specified in the config file use the built-in defaults. See
[docs/USAGE.md](docs/USAGE.md) for the full configuration reference and
[docs/RULES.md](docs/RULES.md) for detailed rule documentation.

## Editor Integration

### Neovim (conform.nvim)

```lua
require("conform").setup({
  formatters_by_ft = {
    make = { "makefmt" },
  },
  formatters = {
    makefmt = {
      command = "makefmt",
      stdin = true,
    },
  },
})
```

### VS Code

Add to `.vscode/settings.json`:

```json
{
  "[makefile]": {
    "editor.formatOnSave": true
  },
  "editor.formatOnSaveMode": "file"
}
```

Then configure a formatter extension (e.g., Run on Save) to execute
`makefmt -w` on Makefiles.

### Pre-commit hook

Add to `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/donaldgifford/makefmt
    rev: v0.1.0
    hooks:
      - id: makefmt
        name: makefmt
        entry: makefmt -w
        language: golang
        types: [makefile]
```

## CI Usage

### GitHub Actions

```yaml
- name: Check Makefile formatting
  run: |
    go install github.com/donaldgifford/makefmt/cmd/makefmt@latest
    makefmt --check Makefile
```

`makefmt --check` exits with code 1 if any file is not formatted, making it
suitable for CI gates.

## Documentation

- [CLI Reference (USAGE.md)](docs/USAGE.md) — flags, config, examples
- [Formatting Rules (RULES.md)](docs/RULES.md) — all rules with before/after examples
- [Design Document (DESIGN.md)](docs/DESIGN.md) — architecture and design decisions

## Development

Requires Go 1.25+ and golangci-lint. Tool versions are managed by
[mise](https://mise.jdx.dev/) (`mise.toml`).

```bash
make build          # Build binary to build/bin/makefmt
make test           # Run all tests with race detector
make lint           # Run golangci-lint
make fmt            # Format with gofmt + goimports
make ci             # Full CI pipeline: lint + test + build
make help           # Show all available targets
```

## License

Apache 2.0 — see [LICENSE](LICENSE).
