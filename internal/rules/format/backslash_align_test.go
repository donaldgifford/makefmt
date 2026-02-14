package format

import (
	"testing"

	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/parser"
)

func TestBackslashAlignFixedColumn(t *testing.T) {
	rule := &BackslashAlign{}
	cfg := &config.DefaultConfig().Formatter // default: backslash_column is 79

	raw := "@if [ -z \"$(TAG)\" ]; then \\\n\techo \"Error\"; \\\n\texit 1; \\\nfi"
	node := &parser.Node{
		Type: parser.NodeRaw,
		Raw:  raw,
	}

	result := rule.Format([]*parser.Node{node}, cfg)
	n := result[0]

	// Verify all continuation lines have backslash at the same column.
	lines := splitTestLines(n.Raw)
	for _, line := range lines {
		if !hasTrailingBackslash(line) {
			continue
		}
		// Backslash should be near column 79.
		bsIdx := lastBackslashIdx(line)
		if bsIdx < 0 {
			t.Error("expected trailing backslash")
		}
	}
}

func TestBackslashAlignAutoColumn(t *testing.T) {
	rule := &BackslashAlign{}
	cfg := &config.DefaultConfig().Formatter
	cfg.BackslashColumn = 0 // auto mode

	raw := "short \\\nlonger content \\\nx \\"
	node := &parser.Node{
		Type: parser.NodeRaw,
		Raw:  raw,
	}

	result := rule.Format([]*parser.Node{node}, cfg)
	n := result[0]

	// All backslashes should be at the same column.
	lines := splitTestLines(n.Raw)
	var bsCols []int
	for _, line := range lines {
		if hasTrailingBackslash(line) {
			bsCols = append(bsCols, lastBackslashIdx(line))
		}
	}

	if len(bsCols) < 2 {
		t.Fatalf("expected at least 2 continuation lines, got %d", len(bsCols))
	}
	for i := 1; i < len(bsCols); i++ {
		if bsCols[i] != bsCols[0] {
			t.Errorf("backslash columns not aligned: %v", bsCols)
			break
		}
	}
}

func TestBackslashAlignDisabled(t *testing.T) {
	rule := &BackslashAlign{}
	cfg := &config.DefaultConfig().Formatter
	cfg.AlignBackslashContinuations = false

	raw := "short \\\nlonger content \\"
	node := &parser.Node{
		Type: parser.NodeRaw,
		Raw:  raw,
	}

	result := rule.Format([]*parser.Node{node}, cfg)
	if result[0].Raw != raw {
		t.Error("disabled rule should not modify nodes")
	}
}

func TestBackslashAlignNoContinuation(t *testing.T) {
	rule := &BackslashAlign{}
	cfg := &config.DefaultConfig().Formatter

	node := &parser.Node{
		Type: parser.NodeComment,
		Raw:  "# just a comment",
	}

	result := rule.Format([]*parser.Node{node}, cfg)
	if result[0].Raw != node.Raw {
		t.Error("node without continuation should not be modified")
	}
}

func splitTestLines(s string) []string {
	var lines []string
	start := 0
	for i := range len(s) {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func hasTrailingBackslash(line string) bool {
	for i := len(line) - 1; i >= 0; i-- {
		if line[i] == '\\' {
			return true
		}
		if line[i] != ' ' && line[i] != '\t' {
			return false
		}
	}
	return false
}

func lastBackslashIdx(line string) int {
	for i := len(line) - 1; i >= 0; i-- {
		if line[i] == '\\' {
			return i
		}
		if line[i] != ' ' && line[i] != '\t' {
			return -1
		}
	}
	return -1
}
