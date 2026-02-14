package format

import (
	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/parser"
)

// FinalNewline ensures the file ends with exactly one newline.
// This is handled by removing trailing blank lines from the AST.
// The writer always appends a newline after the last node, so
// the result is exactly one trailing newline.
type FinalNewline struct{}

// Name returns the config key for this rule.
func (r *FinalNewline) Name() string {
	return "insert_final_newline"
}

// Format removes trailing blank lines so the writer produces exactly
// one final newline.
func (r *FinalNewline) Format(nodes []*parser.Node, cfg *config.FormatterConfig) []*parser.Node {
	if !cfg.InsertFinalNewline {
		return nodes
	}

	if len(nodes) == 0 {
		return nodes
	}

	// Remove trailing blank lines.
	result := make([]*parser.Node, len(nodes))
	copy(result, nodes)

	for len(result) > 0 && result[len(result)-1].Type == parser.NodeBlankLine {
		result = result[:len(result)-1]
	}

	return result
}
