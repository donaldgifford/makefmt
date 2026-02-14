// Package parser provides a line-by-line Makefile parser that produces an AST.
package parser

// NodeType classifies a parsed Makefile line.
type NodeType int

const (
	NodeComment      NodeType = iota // # comment
	NodeSectionHeader                // ##@ Section Name
	NodeBannerComment                // Decorative separators (###..., # ===..., ## box ##)
	NodeBlankLine                    // Empty or whitespace-only line
	NodeAssignment                   // VAR = value, VAR := value, etc.
	NodeRule                         // target: prerequisites
	NodeRecipe                       // \t command (recipe line)
	NodeConditional                  // ifeq/ifdef/ifndef/else/endif
	NodeInclude                      // include, -include, sinclude
	NodeDirective                    // .PHONY, .DEFAULT_GOAL, export, unexport, etc.
	NodeRaw                          // Unparseable lines preserved verbatim (incl. define/endef)
)

//go:generate stringer -type=NodeType

// Node represents a single parsed element in a Makefile AST.
type Node struct {
	Type     NodeType
	Line     int      // 1-indexed source line number.
	Raw      string   // Original text (for diffing and round-tripping).
	Children []*Node  // Recipe lines under a rule, body of conditional.
	Fields   NodeFields
}

// NodeFields holds type-specific parsed data for a Node.
type NodeFields struct {
	// Assignment fields.
	VarName  string
	AssignOp string // =, :=, ::=, ?=, +=, !=
	VarValue string

	// Rule fields.
	Targets       []string
	Prerequisites []string
	OrderOnly     []string // After |
	InlineHelp    string   // "## Description" trailing comment on rule lines.

	// Conditional fields.
	Directive string // ifeq, ifneq, ifdef, ifndef, else, endif.
	Condition string // The condition expression.

	// Include fields.
	IncludeType string   // include, -include, sinclude.
	Paths       []string

	// Comment / SectionHeader / BannerComment fields.
	Text   string
	Inline bool   // Trailing comment on another line.
	Prefix string // "#", "##", "##@" — preserved exactly by the writer.
}

// Clone returns a deep copy of the node.
func (n *Node) Clone() *Node {
	if n == nil {
		return nil
	}

	clone := &Node{
		Type:   n.Type,
		Line:   n.Line,
		Raw:    n.Raw,
		Fields: n.Fields.clone(),
	}

	if n.Children != nil {
		clone.Children = make([]*Node, len(n.Children))
		for i, child := range n.Children {
			clone.Children[i] = child.Clone()
		}
	}

	return clone
}

// clone returns a deep copy of NodeFields.
func (f NodeFields) clone() NodeFields {
	c := f

	c.Targets = cloneStrings(f.Targets)
	c.Prerequisites = cloneStrings(f.Prerequisites)
	c.OrderOnly = cloneStrings(f.OrderOnly)
	c.Paths = cloneStrings(f.Paths)

	return c
}

func cloneStrings(s []string) []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)
	return out
}
