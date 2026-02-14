package format

import (
	"strings"

	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/parser"
)

// Conditional directive keywords.
const (
	directiveElse  = "else"
	directiveEndif = "endif"
)

// ConditionalIndent indents the body of ifeq/ifdef/ifndef blocks.
type ConditionalIndent struct{}

// Name returns the config key for this rule.
func (*ConditionalIndent) Name() string {
	return "indent_conditionals"
}

// Format applies indentation to conditional block bodies.
func (*ConditionalIndent) Format(nodes []*parser.Node, cfg *config.FormatterConfig) []*parser.Node {
	if !cfg.IndentConditionals || cfg.ConditionalIndent <= 0 {
		return nodes
	}

	indent := strings.Repeat(" ", cfg.ConditionalIndent)
	return indentConditionals(nodes, indent, 0)
}

// indentConditionals walks nodes and applies indentation based on
// conditional nesting level.
func indentConditionals(nodes []*parser.Node, indent string, level int) []*parser.Node {
	result := make([]*parser.Node, 0, len(nodes))

	for _, n := range nodes {
		switch {
		case n.Type == parser.NodeConditional && isConditionalOpen(n.Fields.Directive):
			result = append(result, applyIndent(n, indent, level))
			level++

		case n.Type == parser.NodeConditional && n.Fields.Directive == directiveElse:
			result = append(result, applyIndent(n, indent, level-1))

		case n.Type == parser.NodeConditional && n.Fields.Directive == directiveEndif:
			level--
			if level < 0 {
				level = 0
			}
			result = append(result, applyIndent(n, indent, level))

		default:
			if level > 0 {
				result = append(result, applyIndent(n, indent, level))
			} else {
				result = append(result, n)
			}
		}
	}

	return result
}

// isConditionalOpen returns true for directives that open a conditional block.
func isConditionalOpen(directive string) bool {
	switch directive {
	case "ifeq", "ifneq", "ifdef", "ifndef":
		return true
	}
	return false
}

// applyIndent prepends the given indent to the node. If Raw is empty
// (cleared by a prior rule), it reconstructs Raw from fields first.
func applyIndent(n *parser.Node, indent string, level int) *parser.Node {
	if level <= 0 {
		return n
	}

	prefix := strings.Repeat(indent, level)
	clone := n.Clone()

	raw := clone.Raw
	if raw == "" {
		raw = reconstructRaw(clone)
	}
	clone.Raw = prefix + raw

	return clone
}

// reconstructRaw produces the text representation of a node from its
// fields, mirroring what the writer would emit. This is needed when a
// prior rule cleared Raw (e.g., assignment spacing normalizes Raw).
func reconstructRaw(n *parser.Node) string {
	switch n.Type {
	case parser.NodeAssignment:
		s := n.Fields.VarName + " " + n.Fields.AssignOp
		if n.Fields.VarValue != "" {
			s += " " + n.Fields.VarValue
		}
		return s

	case parser.NodeComment:
		if n.Fields.Text != "" {
			return n.Fields.Prefix + " " + n.Fields.Text
		}
		return n.Fields.Prefix

	case parser.NodeConditional:
		if n.Fields.Condition != "" {
			return n.Fields.Directive + " " + n.Fields.Condition
		}
		return n.Fields.Directive

	case parser.NodeInclude:
		if len(n.Fields.Paths) > 0 {
			return n.Fields.IncludeType + " " + strings.Join(n.Fields.Paths, " ")
		}
		return n.Fields.IncludeType

	case parser.NodeRule:
		s := strings.Join(n.Fields.Targets, " ") + ":"
		if len(n.Fields.Prerequisites) > 0 {
			s += " " + strings.Join(n.Fields.Prerequisites, " ")
		}
		if n.Fields.InlineHelp != "" {
			s += " ## " + n.Fields.InlineHelp
		}
		return s

	case parser.NodeBlankLine:
		return ""

	default:
		return n.Fields.Text
	}
}
