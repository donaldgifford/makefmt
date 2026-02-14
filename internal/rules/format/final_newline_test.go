package format

import (
	"testing"

	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/parser"
)

func TestFinalNewline(t *testing.T) {
	rule := &FinalNewline{}
	cfg := &config.DefaultConfig().Formatter

	tests := []struct {
		name     string
		nodes    []*parser.Node
		expected int
	}{
		{
			name:     "no trailing blanks",
			nodes:    []*parser.Node{{Type: parser.NodeComment, Raw: "# comment"}},
			expected: 1,
		},
		{
			name: "one trailing blank",
			nodes: []*parser.Node{
				{Type: parser.NodeComment, Raw: "# comment"},
				{Type: parser.NodeBlankLine, Raw: ""},
			},
			expected: 1,
		},
		{
			name: "multiple trailing blanks",
			nodes: []*parser.Node{
				{Type: parser.NodeComment, Raw: "# comment"},
				{Type: parser.NodeBlankLine, Raw: ""},
				{Type: parser.NodeBlankLine, Raw: ""},
				{Type: parser.NodeBlankLine, Raw: ""},
			},
			expected: 1,
		},
		{
			name:     "empty input",
			nodes:    []*parser.Node{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rule.Format(tt.nodes, cfg)
			if len(result) != tt.expected {
				t.Errorf("want %d nodes, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestFinalNewlineDisabled(t *testing.T) {
	rule := &FinalNewline{}
	cfg := &config.DefaultConfig().Formatter
	cfg.InsertFinalNewline = false

	nodes := []*parser.Node{
		{Type: parser.NodeComment, Raw: "# comment"},
		{Type: parser.NodeBlankLine, Raw: ""},
		{Type: parser.NodeBlankLine, Raw: ""},
	}

	result := rule.Format(nodes, cfg)
	if len(result) != 3 {
		t.Errorf("disabled rule should not modify: got %d nodes", len(result))
	}
}
