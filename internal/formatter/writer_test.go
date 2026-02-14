package formatter

import (
	"testing"

	"github.com/donaldgifford/makefmt/internal/parser"
)

func TestWriteRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "comment",
			input: "# This is a comment\n",
		},
		{
			name:  "double hash comment",
			input: "## Go Variables\n",
		},
		{
			name:  "section header",
			input: "##@ Development\n",
		},
		{
			name:  "blank line",
			input: "\n",
		},
		{
			name:  "simple assignment",
			input: "VAR := value\n",
		},
		{
			name:  "assignment no space",
			input: "VAR:=value\n",
		},
		{
			name:  "rule with prereqs",
			input: "build: main.go utils.go\n",
		},
		{
			name:  "rule with inline help",
			input: "build: ## Build the binary\n",
		},
		{
			name: "rule with recipe",
			input: "build:\n" +
				"\t@echo hello\n" +
				"\t@echo world\n",
		},
		{
			name:  "conditional",
			input: "ifeq ($(OS),Windows_NT)\n",
		},
		{
			name:  "include",
			input: "include foo.mk bar.mk\n",
		},
		{
			name:  "directive",
			input: ".PHONY: build test\n",
		},
		{
			name:  "banner comment",
			input: "###############\n",
		},
		{
			name:  "define block",
			input: "define MY_FUNC\n\t@echo hello\nendef\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := parser.Parse(tt.input)
			output := Write(nodes)
			if output != tt.input {
				t.Errorf("round-trip failed:\nwant: %q\ngot:  %q", tt.input, output)
			}
		})
	}
}

func TestWriteReconstructsFromFields(t *testing.T) {
	// When Raw is cleared, the writer should reconstruct from fields.
	tests := []struct {
		name     string
		node     *parser.Node
		expected string
	}{
		{
			name: "comment from fields",
			node: &parser.Node{
				Type: parser.NodeComment,
				Fields: parser.NodeFields{
					Prefix: "#",
					Text:   "hello",
				},
			},
			expected: "# hello\n",
		},
		{
			name: "assignment from fields",
			node: &parser.Node{
				Type: parser.NodeAssignment,
				Fields: parser.NodeFields{
					VarName:  "FOO",
					AssignOp: ":=",
					VarValue: "bar",
				},
			},
			expected: "FOO := bar\n",
		},
		{
			name: "assignment empty value",
			node: &parser.Node{
				Type: parser.NodeAssignment,
				Fields: parser.NodeFields{
					VarName:  "FOO",
					AssignOp: "=",
				},
			},
			expected: "FOO =\n",
		},
		{
			name: "rule from fields",
			node: &parser.Node{
				Type: parser.NodeRule,
				Fields: parser.NodeFields{
					Targets:       []string{"build"},
					Prerequisites: []string{"main.go"},
					InlineHelp:    "Build it",
				},
			},
			expected: "build: main.go ## Build it\n",
		},
		{
			name: "blank line from fields",
			node: &parser.Node{
				Type: parser.NodeBlankLine,
			},
			expected: "\n",
		},
		{
			name: "conditional from fields",
			node: &parser.Node{
				Type: parser.NodeConditional,
				Fields: parser.NodeFields{
					Directive: "ifdef",
					Condition: "DEBUG",
				},
			},
			expected: "ifdef DEBUG\n",
		},
		{
			name: "include from fields",
			node: &parser.Node{
				Type: parser.NodeInclude,
				Fields: parser.NodeFields{
					IncludeType: "include",
					Paths:       []string{"foo.mk", "bar.mk"},
				},
			},
			expected: "include foo.mk bar.mk\n",
		},
		{
			name: "section header from fields",
			node: &parser.Node{
				Type: parser.NodeSectionHeader,
				Fields: parser.NodeFields{
					Prefix: "##@",
					Text:   "Help",
				},
			},
			expected: "##@ Help\n",
		},
		{
			name: "recipe from fields",
			node: &parser.Node{
				Type: parser.NodeRecipe,
				Fields: parser.NodeFields{
					Text: "@echo hello",
				},
			},
			expected: "\t@echo hello\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := Write([]*parser.Node{tt.node})
			if output != tt.expected {
				t.Errorf("want: %q, got: %q", tt.expected, output)
			}
		})
	}
}

func TestWriteFullMakefile(t *testing.T) {
	input := `# Project
PROJECT_NAME := makefmt

###############
##@ Development

.PHONY: build test

build: ## Build the binary
	@mkdir -p build/bin
	@go build -o build/bin/makefmt ./cmd/makefmt

test: ## Run tests
	@go test -v -race ./...

log-%:
	@grep -h -E '^$*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN { FS = ":.*?## " }; { printf "\033[36m==> %s\033[0m\n", $$2 }'
`

	nodes := parser.Parse(input)
	output := Write(nodes)

	if output != input {
		t.Errorf("full Makefile round-trip failed.\nInput length: %d\nOutput length: %d", len(input), len(output))
		logFirstDifference(t, input, output)
	}
}

func logFirstDifference(t *testing.T, input, output string) {
	t.Helper()
	for i := range min(len(input), len(output)) {
		if input[i] == output[i] {
			continue
		}
		start := max(i-20, 0)
		end := min(i+20, len(input))
		t.Errorf("first difference at byte %d:\ninput:  %q\noutput: %q",
			i, input[start:end], output[start:min(end, len(output))])
		return
	}
}
