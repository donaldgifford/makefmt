package format

import (
	"testing"

	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/parser"
)

func TestTrailingWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected string
	}{
		{"trailing spaces", "VAR := value   ", "VAR := value"},
		{"trailing tab", "VAR := value\t", "VAR := value"},
		{"no trailing", "VAR := value", "VAR := value"},
		{"empty", "", ""},
	}

	rule := &TrailingWhitespace{}
	cfg := &config.DefaultConfig().Formatter

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &parser.Node{
				Type: parser.NodeAssignment,
				Raw:  tt.raw,
			}
			result := rule.Format([]*parser.Node{node}, cfg)
			if len(result) != 1 {
				t.Fatalf("expected 1 node, got %d", len(result))
			}
			if result[0].Raw != tt.expected {
				t.Errorf("Raw: want %q, got %q", tt.expected, result[0].Raw)
			}
		})
	}
}

func TestTrailingWhitespaceDisabled(t *testing.T) {
	rule := &TrailingWhitespace{}
	cfg := &config.DefaultConfig().Formatter
	cfg.TrimTrailingWhitespace = false

	node := &parser.Node{
		Type: parser.NodeComment,
		Raw:  "# comment   ",
	}

	result := rule.Format([]*parser.Node{node}, cfg)
	if result[0].Raw != "# comment   " {
		t.Errorf("disabled rule should not modify: got %q", result[0].Raw)
	}
}

func TestTrailingWhitespaceDoesNotMutateInput(t *testing.T) {
	rule := &TrailingWhitespace{}
	cfg := &config.DefaultConfig().Formatter

	original := &parser.Node{
		Type: parser.NodeComment,
		Raw:  "# comment   ",
	}

	rule.Format([]*parser.Node{original}, cfg)

	if original.Raw != "# comment   " {
		t.Error("rule mutated input node")
	}
}
