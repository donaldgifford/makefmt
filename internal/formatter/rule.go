package formatter

import (
	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/parser"
)

// FormatRule transforms AST nodes. Rules are applied in registered order.
type FormatRule interface {
	// Name returns the config key for this rule (e.g., "trim_trailing_whitespace").
	Name() string

	// Format receives the full AST and config, returns a modified AST.
	// Rules should not mutate the input; return new/cloned nodes where
	// changes are needed.
	Format(nodes []*parser.Node, cfg *config.FormatterConfig) []*parser.Node
}
