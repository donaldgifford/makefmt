// Package format contains individual formatting rule implementations.
package format

import (
	"strings"

	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/parser"
)

// TrailingWhitespace removes trailing spaces and tabs from every line.
type TrailingWhitespace struct{}

// Name returns the config key for this rule.
func (r *TrailingWhitespace) Name() string {
	return "trim_trailing_whitespace"
}

// Format strips trailing whitespace from all nodes.
func (r *TrailingWhitespace) Format(nodes []*parser.Node, cfg *config.FormatterConfig) []*parser.Node {
	if !cfg.TrimTrailingWhitespace {
		return nodes
	}

	result := make([]*parser.Node, len(nodes))
	for i, n := range nodes {
		result[i] = trimNode(n)
	}
	return result
}

func trimNode(n *parser.Node) *parser.Node {
	clone := n.Clone()

	// Trim the Raw field (may be multi-line for continuation blocks).
	if clone.Raw != "" {
		clone.Raw = trimRawLines(clone.Raw)
	}

	// Trim text-bearing fields.
	clone.Fields.Text = strings.TrimRight(clone.Fields.Text, " \t")
	clone.Fields.VarValue = strings.TrimRight(clone.Fields.VarValue, " \t")
	clone.Fields.InlineHelp = strings.TrimRight(clone.Fields.InlineHelp, " \t")
	clone.Fields.Condition = strings.TrimRight(clone.Fields.Condition, " \t")

	// Recurse into children.
	for i, child := range clone.Children {
		clone.Children[i] = trimNode(child)
	}

	return clone
}

// trimRawLines trims trailing whitespace from each line in a
// potentially multi-line Raw field.
func trimRawLines(raw string) string {
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}
