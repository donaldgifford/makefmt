package format

import (
	"strings"
	"testing"

	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/parser"
)

func TestConditionalIndentSimple(t *testing.T) {
	rule := &ConditionalIndent{}
	cfg := &config.DefaultConfig().Formatter // IndentConditionals=true, ConditionalIndent=2

	nodes := []*parser.Node{
		{Type: parser.NodeConditional, Raw: "ifeq ($(OS),Linux)", Fields: parser.NodeFields{Directive: "ifeq", Condition: "($(OS),Linux)"}},
		{Type: parser.NodeAssignment, Raw: "CC := gcc"},
		{Type: parser.NodeConditional, Raw: "endif", Fields: parser.NodeFields{Directive: "endif"}},
	}

	result := rule.Format(nodes, cfg)

	// ifeq should not be indented.
	if result[0].Raw != "ifeq ($(OS),Linux)" {
		t.Errorf("ifeq: got %q", result[0].Raw)
	}
	// Body should be indented by 2 spaces.
	if !strings.HasPrefix(result[1].Raw, "  ") {
		t.Errorf("body should be indented: got %q", result[1].Raw)
	}
	// endif should not be indented.
	if result[2].Raw != "endif" {
		t.Errorf("endif: got %q", result[2].Raw)
	}
}

func TestConditionalIndentNested(t *testing.T) {
	rule := &ConditionalIndent{}
	cfg := &config.DefaultConfig().Formatter

	nodes := []*parser.Node{
		{Type: parser.NodeConditional, Raw: "ifdef DEBUG", Fields: parser.NodeFields{Directive: "ifdef", Condition: "DEBUG"}},
		{Type: parser.NodeConditional, Raw: "ifeq ($(OS),Linux)", Fields: parser.NodeFields{Directive: "ifeq", Condition: "($(OS),Linux)"}},
		{Type: parser.NodeAssignment, Raw: "CC := gcc"},
		{Type: parser.NodeConditional, Raw: "endif", Fields: parser.NodeFields{Directive: "endif"}},
		{Type: parser.NodeConditional, Raw: "endif", Fields: parser.NodeFields{Directive: "endif"}},
	}

	result := rule.Format(nodes, cfg)

	// Outer ifdef: no indent.
	if result[0].Raw != "ifdef DEBUG" {
		t.Errorf("outer ifdef: got %q", result[0].Raw)
	}
	// Inner ifeq: 2 spaces (level 1).
	if !strings.HasPrefix(result[1].Raw, "  ") || strings.HasPrefix(result[1].Raw, "    ") {
		t.Errorf("inner ifeq should be at level 1 (2 spaces): got %q", result[1].Raw)
	}
	// Body inside nested: 4 spaces (level 2).
	if !strings.HasPrefix(result[2].Raw, "    ") {
		t.Errorf("nested body should be at level 2 (4 spaces): got %q", result[2].Raw)
	}
	// Inner endif: 2 spaces (level 1).
	if !strings.HasPrefix(result[3].Raw, "  ") || strings.HasPrefix(result[3].Raw, "    ") {
		t.Errorf("inner endif should be at level 1: got %q", result[3].Raw)
	}
	// Outer endif: no indent.
	if result[4].Raw != "endif" {
		t.Errorf("outer endif: got %q", result[4].Raw)
	}
}

func TestConditionalIndentElse(t *testing.T) {
	rule := &ConditionalIndent{}
	cfg := &config.DefaultConfig().Formatter

	nodes := []*parser.Node{
		{Type: parser.NodeConditional, Raw: "ifdef DEBUG", Fields: parser.NodeFields{Directive: "ifdef", Condition: "DEBUG"}},
		{Type: parser.NodeAssignment, Raw: "CFLAGS := -g"},
		{Type: parser.NodeConditional, Raw: "else", Fields: parser.NodeFields{Directive: "else"}},
		{Type: parser.NodeAssignment, Raw: "CFLAGS := -O2"},
		{Type: parser.NodeConditional, Raw: "endif", Fields: parser.NodeFields{Directive: "endif"}},
	}

	result := rule.Format(nodes, cfg)

	// else aligns with ifdef (no indent).
	if result[2].Raw != "else" {
		t.Errorf("else should align with ifdef: got %q", result[2].Raw)
	}
	// Body after else should be indented.
	if !strings.HasPrefix(result[3].Raw, "  ") {
		t.Errorf("body after else should be indented: got %q", result[3].Raw)
	}
}

func TestConditionalIndentDisabled(t *testing.T) {
	rule := &ConditionalIndent{}
	cfg := &config.DefaultConfig().Formatter
	cfg.IndentConditionals = false

	nodes := []*parser.Node{
		{Type: parser.NodeConditional, Raw: "ifdef DEBUG", Fields: parser.NodeFields{Directive: "ifdef", Condition: "DEBUG"}},
		{Type: parser.NodeAssignment, Raw: "CC := gcc"},
		{Type: parser.NodeConditional, Raw: "endif", Fields: parser.NodeFields{Directive: "endif"}},
	}

	result := rule.Format(nodes, cfg)

	if result[1].Raw != "CC := gcc" {
		t.Error("disabled rule should not indent")
	}
}
