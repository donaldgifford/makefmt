package format

import (
	"strings"

	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/parser"
)

// BackslashAlign aligns trailing backslashes in continuation blocks
// to a consistent column.
type BackslashAlign struct{}

// Name returns the config key for this rule.
func (*BackslashAlign) Name() string {
	return "align_backslash_continuations"
}

// Format aligns trailing backslashes in continuation lines.
func (*BackslashAlign) Format(nodes []*parser.Node, cfg *config.FormatterConfig) []*parser.Node {
	if !cfg.AlignBackslashContinuations {
		return nodes
	}

	result := make([]*parser.Node, len(nodes))
	copy(result, nodes)

	for i, n := range result {
		if !hasContinuation(n.Raw) {
			continue
		}

		// Process each continuation block in the node's Raw field.
		result[i] = alignBackslashes(n, cfg.BackslashColumn)
	}

	return result
}

// hasContinuation returns true if the raw text contains a line ending
// with backslash (continuation).
func hasContinuation(raw string) bool {
	for line := range strings.SplitSeq(raw, "\n") {
		trimmed := strings.TrimRight(line, " \t")
		if strings.HasSuffix(trimmed, "\\") {
			return true
		}
	}
	return false
}

// alignBackslashes clones the node and aligns all trailing backslashes
// in its Raw field to the target column.
func alignBackslashes(n *parser.Node, backslashCol int) *parser.Node {
	clone := n.Clone()
	lines := strings.Split(clone.Raw, "\n")

	// Find the continuation block: all lines that end with \.
	// Also find the max content width for auto-column mode.
	maxContentWidth := 0
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if !strings.HasSuffix(trimmed, "\\") {
			continue
		}
		// Content width = everything before the trailing backslash.
		content := strings.TrimRight(trimmed[:len(trimmed)-1], " \t")
		if len(content) > maxContentWidth {
			maxContentWidth = len(content)
		}
	}

	// Determine the target column.
	targetCol := backslashCol
	if targetCol == 0 {
		// Auto mode: longest content + 1 space + backslash.
		targetCol = maxContentWidth + 2
	}

	// Align each continuation line.
	for i, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if !strings.HasSuffix(trimmed, "\\") {
			continue
		}

		content := strings.TrimRight(trimmed[:len(trimmed)-1], " \t")
		// Pad content to targetCol - 1 (the backslash goes at targetCol).
		padWidth := max(targetCol-1-len(content), 1) // Always at least one space before \.
		lines[i] = content + strings.Repeat(" ", padWidth) + "\\"
	}

	clone.Raw = strings.Join(lines, "\n")
	return clone
}
