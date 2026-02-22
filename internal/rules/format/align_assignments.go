package format

import (
	"strings"

	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/parser"
)

// AlignAssignments column-aligns assignment operators within groups of
// consecutive assignment lines. Groups are delimited by blank lines,
// comments, or any non-assignment node. Existing over-padding is
// normalized down to the minimum column required by the group.
type AlignAssignments struct{}

// Name returns the config key for this rule.
func (*AlignAssignments) Name() string {
	return "align_assignments"
}

// Format aligns assignment operators in consecutive assignment groups.
func (*AlignAssignments) Format(nodes []*parser.Node, cfg *config.FormatterConfig) []*parser.Node {
	if !cfg.AlignAssignments {
		return nodes
	}

	result := make([]*parser.Node, len(nodes))
	copy(result, nodes)

	i := 0
	for i < len(result) {
		if result[i].Type != parser.NodeAssignment {
			i++
			continue
		}

		// Collect all consecutive assignments starting at i.
		start := i
		for i < len(result) && result[i].Type == parser.NodeAssignment {
			i++
		}

		// Single-assignment groups need no padding.
		if i-start > 1 {
			alignGroup(result[start:i], cfg.AssignmentSpacing)
		}
	}

	return result
}

// alignGroup pads the VarName of each node in the group so that all
// assignment operators start at the same column. The column is determined
// by the longest bare (untrimmed) VarName in the group.
func alignGroup(group []*parser.Node, spacingMode string) {
	// Measure bare name lengths — trim any padding from a prior run to
	// guarantee idempotent output.
	maxLen := 0
	for _, n := range group {
		name := strings.TrimRight(n.Fields.VarName, " ")
		if l := len(name); l > maxLen {
			maxLen = l
		}
	}

	for i, n := range group {
		clone := n.Clone()
		name := strings.TrimRight(clone.Fields.VarName, " ")
		padded := name + strings.Repeat(" ", maxLen-len(name))

		switch spacingMode {
		case "no_space":
			// Build Raw directly: "PADDED_NAME:=value" (no spaces around op).
			raw := padded + clone.Fields.AssignOp
			if clone.Fields.VarValue != "" {
				raw += clone.Fields.VarValue
			}
			clone.Raw = raw
		default:
			// "space" or "preserve" — clear Raw so the writer reconstructs
			// as "PADDED_NAME := value" (writer always adds spaces).
			clone.Fields.VarName = padded
			clone.Raw = ""
		}

		group[i] = clone
	}
}
