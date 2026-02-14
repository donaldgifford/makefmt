package format

import (
	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/parser"
)

// BlankLines collapses consecutive blank lines down to the configured maximum.
type BlankLines struct{}

// Name returns the config key for this rule.
func (*BlankLines) Name() string {
	return "max_blank_lines"
}

// Format collapses runs of blank lines to at most cfg.MaxBlankLines.
func (*BlankLines) Format(nodes []*parser.Node, cfg *config.FormatterConfig) []*parser.Node {
	if cfg.MaxBlankLines < 0 {
		return nodes
	}

	result := make([]*parser.Node, 0, len(nodes))
	blankCount := 0

	for _, n := range nodes {
		if n.Type == parser.NodeBlankLine {
			blankCount++
			if blankCount <= cfg.MaxBlankLines {
				result = append(result, n)
			}
		} else {
			blankCount = 0
			result = append(result, n)
		}
	}

	return result
}
