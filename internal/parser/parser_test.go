package parser

import (
	"testing"
)

func TestParseEmpty(t *testing.T) {
	nodes := Parse("")
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes for empty input, got %d", len(nodes))
	}
}

func TestParseBlankOnly(t *testing.T) {
	nodes := Parse("\n\n\n")
	for _, n := range nodes {
		if n.Type != NodeBlankLine {
			t.Errorf("expected NodeBlankLine, got %v", n.Type)
		}
	}
}

func TestClassifyComment(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		prefix string
		text   string
	}{
		{
			name:   "single hash",
			input:  "# This is a comment",
			prefix: "#",
			text:   "This is a comment",
		},
		{
			name:   "double hash",
			input:  "## Go Variables",
			prefix: "##",
			text:   "Go Variables",
		},
		{
			name:   "empty comment",
			input:  "#",
			prefix: "#",
			text:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := Parse(tt.input)
			if len(nodes) != 1 {
				t.Fatalf("expected 1 node, got %d", len(nodes))
			}
			n := nodes[0]
			if n.Type != NodeComment {
				t.Errorf("expected NodeComment, got %v", n.Type)
			}
			if n.Fields.Prefix != tt.prefix {
				t.Errorf("prefix: want %q, got %q", tt.prefix, n.Fields.Prefix)
			}
			if n.Fields.Text != tt.text {
				t.Errorf("text: want %q, got %q", tt.text, n.Fields.Text)
			}
		})
	}
}

func TestClassifySectionHeader(t *testing.T) {
	nodes := Parse("##@ Development")
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	n := nodes[0]
	if n.Type != NodeSectionHeader {
		t.Errorf("expected NodeSectionHeader, got %v", n.Type)
	}
	if n.Fields.Text != "Development" {
		t.Errorf("text: want %q, got %q", "Development", n.Fields.Text)
	}
	if n.Fields.Prefix != "##@" {
		t.Errorf("prefix: want %q, got %q", "##@", n.Fields.Prefix)
	}
}

func TestClassifyBannerComment(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"hash line", "###############"},
		{"equals separator", "# ============================================================================="},
		{"box style", "## Self-Documenting Makefile Help                                     ##"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := Parse(tt.input)
			if len(nodes) != 1 {
				t.Fatalf("expected 1 node, got %d", len(nodes))
			}
			if nodes[0].Type != NodeBannerComment {
				t.Errorf("expected NodeBannerComment, got %v", nodes[0].Type)
			}
		})
	}
}

func TestClassifyAssignment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		varName  string
		assignOp string
		varValue string
	}{
		{"simple equals", "VAR = value", "VAR", "=", "value"},
		{"colon equals", "VAR := value", "VAR", ":=", "value"},
		{"double colon equals", "VAR ::= value", "VAR", "::=", "value"},
		{"question equals", "VAR ?= value", "VAR", "?=", "value"},
		{"plus equals", "VAR += value", "VAR", "+=", "value"},
		{"shell equals", "VAR != value", "VAR", "!=", "value"},
		{"no space", "VAR:=value", "VAR", ":=", "value"},
		{"complex value", "GO_PACKAGE := github.com/$(PROJECT_OWNER)/$(PROJECT_NAME)", "GO_PACKAGE", ":=", "github.com/$(PROJECT_OWNER)/$(PROJECT_NAME)"},
		{"empty value", "VAR =", "VAR", "=", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := Parse(tt.input)
			if len(nodes) != 1 {
				t.Fatalf("expected 1 node, got %d", len(nodes))
			}
			n := nodes[0]
			if n.Type != NodeAssignment {
				t.Errorf("expected NodeAssignment, got %v", n.Type)
				return
			}
			if n.Fields.VarName != tt.varName {
				t.Errorf("VarName: want %q, got %q", tt.varName, n.Fields.VarName)
			}
			if n.Fields.AssignOp != tt.assignOp {
				t.Errorf("AssignOp: want %q, got %q", tt.assignOp, n.Fields.AssignOp)
			}
			if n.Fields.VarValue != tt.varValue {
				t.Errorf("VarValue: want %q, got %q", tt.varValue, n.Fields.VarValue)
			}
		})
	}
}

func TestClassifyRule(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		targets       []string
		prerequisites []string
		inlineHelp    string
	}{
		{
			name:    "simple rule",
			input:   "build:",
			targets: []string{"build"},
		},
		{
			name:          "rule with prereqs",
			input:         "build: main.go utils.go",
			targets:       []string{"build"},
			prerequisites: []string{"main.go", "utils.go"},
		},
		{
			name:       "rule with inline help",
			input:      "build: ## Build the binary",
			targets:    []string{"build"},
			inlineHelp: "Build the binary",
		},
		{
			name:          "rule with prereqs and help",
			input:         "ci: lint test build ## Run CI pipeline",
			targets:       []string{"ci"},
			prerequisites: []string{"lint", "test", "build"},
			inlineHelp:    "Run CI pipeline",
		},
		{
			name:    "pattern rule",
			input:   "log-%:",
			targets: []string{"log-%"},
		},
		{
			name:    "catch-all pattern",
			input:   "%:",
			targets: []string{"%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := Parse(tt.input)
			if len(nodes) != 1 {
				t.Fatalf("expected 1 node, got %d", len(nodes))
			}
			n := nodes[0]
			if n.Type != NodeRule {
				t.Errorf("expected NodeRule, got %v", n.Type)
				return
			}
			if !slicesEqual(n.Fields.Targets, tt.targets) {
				t.Errorf("Targets: want %v, got %v", tt.targets, n.Fields.Targets)
			}
			if !slicesEqual(n.Fields.Prerequisites, tt.prerequisites) {
				t.Errorf("Prerequisites: want %v, got %v", tt.prerequisites, n.Fields.Prerequisites)
			}
			if n.Fields.InlineHelp != tt.inlineHelp {
				t.Errorf("InlineHelp: want %q, got %q", tt.inlineHelp, n.Fields.InlineHelp)
			}
		})
	}
}

func TestClassifyRecipe(t *testing.T) {
	input := "build:\n\t@echo hello\n\t@echo world"
	nodes := Parse(input)

	if len(nodes) != 1 {
		t.Fatalf("expected 1 top-level node, got %d", len(nodes))
	}

	rule := nodes[0]
	if rule.Type != NodeRule {
		t.Fatalf("expected NodeRule, got %v", rule.Type)
	}

	if len(rule.Children) != 2 {
		t.Fatalf("expected 2 recipe children, got %d", len(rule.Children))
	}

	for _, child := range rule.Children {
		if child.Type != NodeRecipe {
			t.Errorf("expected NodeRecipe child, got %v", child.Type)
		}
	}
}

func TestClassifyConditional(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		directive string
		condition string
	}{
		{"ifeq", "ifeq ($(OS),Windows_NT)", "ifeq", "($(OS),Windows_NT)"},
		{"ifdef", "ifdef DEBUG", "ifdef", "DEBUG"},
		{"ifndef", "ifndef CC", "ifndef", "CC"},
		{"else", "else", "else", ""},
		{"endif", "endif", "endif", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := Parse(tt.input)
			if len(nodes) != 1 {
				t.Fatalf("expected 1 node, got %d", len(nodes))
			}
			n := nodes[0]
			if n.Type != NodeConditional {
				t.Errorf("expected NodeConditional, got %v", n.Type)
				return
			}
			if n.Fields.Directive != tt.directive {
				t.Errorf("Directive: want %q, got %q", tt.directive, n.Fields.Directive)
			}
			if n.Fields.Condition != tt.condition {
				t.Errorf("Condition: want %q, got %q", tt.condition, n.Fields.Condition)
			}
		})
	}
}

func TestClassifyInclude(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		includeType string
		paths       []string
	}{
		{"include", "include foo.mk bar.mk", "include", []string{"foo.mk", "bar.mk"}},
		{"dash include", "-include optional.mk", "-include", []string{"optional.mk"}},
		{"sinclude", "sinclude optional.mk", "sinclude", []string{"optional.mk"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := Parse(tt.input)
			if len(nodes) != 1 {
				t.Fatalf("expected 1 node, got %d", len(nodes))
			}
			n := nodes[0]
			if n.Type != NodeInclude {
				t.Errorf("expected NodeInclude, got %v", n.Type)
				return
			}
			if n.Fields.IncludeType != tt.includeType {
				t.Errorf("IncludeType: want %q, got %q", tt.includeType, n.Fields.IncludeType)
			}
			if !slicesEqual(n.Fields.Paths, tt.paths) {
				t.Errorf("Paths: want %v, got %v", tt.paths, n.Fields.Paths)
			}
		})
	}
}

func TestClassifyDirective(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"phony", ".PHONY: build test"},
		{"export", "export PATH"},
		{"unexport", "unexport SECRET"},
		{"default goal", ".DEFAULT_GOAL := help"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := Parse(tt.input)
			if len(nodes) != 1 {
				t.Fatalf("expected 1 node, got %d", len(nodes))
			}
			n := nodes[0]
			if n.Type != NodeDirective {
				t.Errorf("expected NodeDirective, got %v for input %q", n.Type, tt.input)
			}
		})
	}
}

func TestDefineBlock(t *testing.T) {
	input := "define MY_FUNC\n\t@echo hello\n\t@echo world\nendef"
	nodes := Parse(input)

	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}

	n := nodes[0]
	if n.Type != NodeRaw {
		t.Errorf("expected NodeRaw for define block, got %v", n.Type)
	}
	if n.Raw != input {
		t.Errorf("raw: want %q, got %q", input, n.Raw)
	}
}

func TestContinuationLines(t *testing.T) {
	input := "VAR = one \\\ntwo \\\nthree"
	nodes := Parse(input)

	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}

	n := nodes[0]
	if n.Type != NodeAssignment {
		t.Errorf("expected NodeAssignment, got %v", n.Type)
	}
	// The raw field should contain the original multi-line text.
	if n.Raw != input {
		t.Errorf("raw should preserve original text:\nwant: %q\ngot:  %q", input, n.Raw)
	}
}

func TestRawPreservesOriginalText(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"comment", "# A comment"},
		{"assignment", "VAR := value"},
		{"blank", " "},
		{"section header", "##@ Section"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := Parse(tt.input)
			if len(nodes) != 1 {
				t.Fatalf("expected 1 node, got %d", len(nodes))
			}
			if nodes[0].Raw != tt.input {
				t.Errorf("Raw: want %q, got %q", tt.input, nodes[0].Raw)
			}
		})
	}
}

func TestLineNumbers(t *testing.T) {
	input := "# comment\nVAR := val\n\nbuild:\n\t@echo hi"
	nodes := Parse(input)

	expected := []struct {
		line     int
		nodeType NodeType
	}{
		{1, NodeComment},
		{2, NodeAssignment},
		{3, NodeBlankLine},
		{4, NodeRule},
	}

	if len(nodes) != len(expected) {
		t.Fatalf("expected %d top-level nodes, got %d", len(expected), len(nodes))
	}

	for i, exp := range expected {
		if nodes[i].Line != exp.line {
			t.Errorf("node %d: line want %d, got %d", i, exp.line, nodes[i].Line)
		}
		if nodes[i].Type != exp.nodeType {
			t.Errorf("node %d: type want %v, got %v", i, exp.nodeType, nodes[i].Type)
		}
	}

	// Recipe should be child of rule.
	if len(nodes[3].Children) != 1 {
		t.Fatalf("expected 1 recipe child, got %d", len(nodes[3].Children))
	}
	if nodes[3].Children[0].Line != 5 {
		t.Errorf("recipe line: want 5, got %d", nodes[3].Children[0].Line)
	}
}

func TestParseExampleMakefile(t *testing.T) {
	// Test with a realistic Makefile excerpt to verify integration.
	input := `# Project Variables

PROJECT_NAME := my-project
PROJECT_OWNER := donaldgifford

###############
##@ Development

.PHONY: build test

build: ## Build the binary
	@ $(MAKE) --no-print-directory log-$@
	@mkdir -p $(BIN_DIR)

test: ## Run tests
	@go test -v -race ./...

ifeq ($(OS),Windows_NT)
CC = cl
else
CC = gcc
endif

log-%:
	@grep -h -E '^$*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN { FS = ":.*?## " }; { printf "\033[36m==> %s\033[0m\n", $$2 }'
`

	nodes := Parse(input)
	if len(nodes) == 0 {
		t.Fatal("expected nodes from parsing example Makefile")
	}

	// Verify key node types are present.
	typeCount := make(map[NodeType]int)
	for _, n := range nodes {
		typeCount[n.Type]++
	}

	if typeCount[NodeComment] < 1 {
		t.Error("expected at least 1 comment node")
	}
	if typeCount[NodeAssignment] < 2 {
		t.Error("expected at least 2 assignment nodes")
	}
	if typeCount[NodeBannerComment] < 1 {
		t.Error("expected at least 1 banner comment node")
	}
	if typeCount[NodeSectionHeader] < 1 {
		t.Error("expected at least 1 section header node")
	}
	if typeCount[NodeRule] < 2 {
		t.Error("expected at least 2 rule nodes")
	}
	if typeCount[NodeConditional] < 1 {
		t.Error("expected at least 1 conditional node")
	}
}

func TestNodeClone(t *testing.T) {
	original := &Node{
		Type: NodeRule,
		Line: 1,
		Raw:  "build: main.go",
		Fields: NodeFields{
			Targets:       []string{"build"},
			Prerequisites: []string{"main.go"},
		},
		Children: []*Node{
			{
				Type: NodeRecipe,
				Line: 2,
				Raw:  "\t@echo hello",
			},
		},
	}

	clone := original.Clone()

	// Verify clone is a deep copy.
	if clone == original {
		t.Error("clone should be a different pointer")
	}
	if clone.Children[0] == original.Children[0] {
		t.Error("children should be deep copied")
	}

	// Mutating clone should not affect original.
	clone.Fields.Targets[0] = "test"
	if original.Fields.Targets[0] == "test" {
		t.Error("mutating clone affected original targets")
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
