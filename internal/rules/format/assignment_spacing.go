package format

import (
	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/parser"
)

// AssignmentSpacing normalizes whitespace around assignment operators.
type AssignmentSpacing struct{}

// Name returns the config key for this rule.
func (*AssignmentSpacing) Name() string {
	return "assignment_spacing"
}

// Format normalizes spacing around assignment operators based on config.
func (*AssignmentSpacing) Format(nodes []*parser.Node, cfg *config.FormatterConfig) []*parser.Node {
	if cfg.AssignmentSpacing == "preserve" {
		return nodes
	}

	result := make([]*parser.Node, len(nodes))
	for i, n := range nodes {
		if n.Type == parser.NodeAssignment {
			result[i] = normalizeAssignment(n, cfg.AssignmentSpacing)
		} else {
			result[i] = n
		}
	}
	return result
}

func normalizeAssignment(n *parser.Node, mode string) *parser.Node {
	clone := n.Clone()

	// Clear Raw so the writer reconstructs from fields with proper spacing.
	// The writer always emits "VarName <op> VarValue" with spaces.
	switch mode {
	case "space":
		// Writer default is "VAR := val" (space around operator).
		// Fields are already parsed without spacing, so clearing Raw suffices.
		clone.Raw = ""
	case "no_space":
		// Reconstruct as "VAR:=val" — we need to set Raw directly since
		// the writer's default includes spaces.
		raw := clone.Fields.VarName + clone.Fields.AssignOp
		if clone.Fields.VarValue != "" {
			raw += clone.Fields.VarValue
		}
		clone.Raw = raw
	}

	return clone
}
