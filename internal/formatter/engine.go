package formatter

import (
	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/parser"
)

// Run applies each formatting rule in order, piping the output of one
// as input to the next.
func Run(nodes []*parser.Node, cfg *config.FormatterConfig, rules []FormatRule) []*parser.Node {
	result := nodes
	for _, rule := range rules {
		result = rule.Format(result, cfg)
	}
	return result
}
