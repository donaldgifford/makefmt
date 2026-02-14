package format

import (
	"testing"

	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/parser"
)

func TestBlankLines(t *testing.T) {
	rule := &BlankLines{}
	cfg := &config.DefaultConfig().Formatter // MaxBlankLines = 2

	tests := []struct {
		name          string
		blankCount    int
		maxBlank      int
		expectedCount int
	}{
		{"1 blank (max 2)", 1, 2, 1},
		{"2 blanks (max 2)", 2, 2, 2},
		{"3 blanks (max 2)", 3, 2, 2},
		{"5 blanks (max 2)", 5, 2, 2},
		{"3 blanks (max 1)", 3, 1, 1},
		{"0 blanks", 0, 2, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.MaxBlankLines = tt.maxBlank

			nodes := make([]*parser.Node, 0)
			nodes = append(nodes, &parser.Node{Type: parser.NodeComment, Raw: "# before"})
			for range tt.blankCount {
				nodes = append(nodes, &parser.Node{Type: parser.NodeBlankLine, Raw: ""})
			}
			nodes = append(nodes, &parser.Node{Type: parser.NodeComment, Raw: "# after"})

			result := rule.Format(nodes, cfg)

			// Count blank lines in result.
			blanks := 0
			for _, n := range result {
				if n.Type == parser.NodeBlankLine {
					blanks++
				}
			}

			if blanks != tt.expectedCount {
				t.Errorf("want %d blank lines, got %d", tt.expectedCount, blanks)
			}
		})
	}
}
