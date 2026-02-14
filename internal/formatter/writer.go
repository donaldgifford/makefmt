// Package formatter provides the formatting engine, writer, and rule interface.
package formatter

import (
	"strings"

	"github.com/donaldgifford/makefmt/internal/parser"
)

// Write serializes an AST back into Makefile text.
//
// For round-trip fidelity, nodes with a non-empty Raw field emit their Raw
// text verbatim. When a formatting rule modifies a node, it should clear or
// update the Raw field so the writer reconstructs from parsed fields instead.
func Write(nodes []*parser.Node) string {
	var b strings.Builder

	for _, n := range nodes {
		writeNode(&b, n)
		b.WriteByte('\n')
	}

	return b.String()
}

func writeNode(b *strings.Builder, n *parser.Node) {
	// If Raw is set, use it for verbatim round-tripping.
	if n.Raw != "" {
		b.WriteString(n.Raw)
		writeChildren(b, n)
		return
	}

	// Reconstruct from parsed fields.
	switch n.Type {
	case parser.NodeBlankLine:
		// Empty line — nothing to write (the trailing \n is added by Write).

	case parser.NodeComment:
		writeComment(b, n)

	case parser.NodeSectionHeader:
		writeSectionHeader(b, n)

	case parser.NodeBannerComment:
		b.WriteString(n.Fields.Text)

	case parser.NodeAssignment:
		writeAssignment(b, n)

	case parser.NodeRule:
		writeRule(b, n)

	case parser.NodeRecipe:
		b.WriteByte('\t')
		b.WriteString(n.Fields.Text)

	case parser.NodeConditional:
		writeConditional(b, n)

	case parser.NodeInclude:
		writeInclude(b, n)

	case parser.NodeDirective:
		b.WriteString(n.Fields.Text)

	case parser.NodeRaw:
		// Should be handled by Raw check above; fallback.
		b.WriteString(n.Fields.Text)
	}

	writeChildren(b, n)
}

func writeChildren(b *strings.Builder, n *parser.Node) {
	for _, child := range n.Children {
		b.WriteByte('\n')
		writeNode(b, child)
	}
}

func writeComment(b *strings.Builder, n *parser.Node) {
	b.WriteString(n.Fields.Prefix)
	if n.Fields.Text != "" {
		b.WriteByte(' ')
		b.WriteString(n.Fields.Text)
	}
}

func writeSectionHeader(b *strings.Builder, n *parser.Node) {
	b.WriteString(n.Fields.Prefix)
	if n.Fields.Text != "" {
		b.WriteByte(' ')
		b.WriteString(n.Fields.Text)
	}
}

func writeAssignment(b *strings.Builder, n *parser.Node) {
	b.WriteString(n.Fields.VarName)
	b.WriteByte(' ')
	b.WriteString(n.Fields.AssignOp)
	if n.Fields.VarValue != "" {
		b.WriteByte(' ')
		b.WriteString(n.Fields.VarValue)
	}
}

func writeRule(b *strings.Builder, n *parser.Node) {
	b.WriteString(strings.Join(n.Fields.Targets, " "))
	b.WriteByte(':')

	if len(n.Fields.Prerequisites) > 0 {
		b.WriteByte(' ')
		b.WriteString(strings.Join(n.Fields.Prerequisites, " "))
	}

	if len(n.Fields.OrderOnly) > 0 {
		b.WriteString(" | ")
		b.WriteString(strings.Join(n.Fields.OrderOnly, " "))
	}

	if n.Fields.InlineHelp != "" {
		b.WriteString(" ## ")
		b.WriteString(n.Fields.InlineHelp)
	}
}

func writeConditional(b *strings.Builder, n *parser.Node) {
	b.WriteString(n.Fields.Directive)
	if n.Fields.Condition != "" {
		b.WriteByte(' ')
		b.WriteString(n.Fields.Condition)
	}
}

func writeInclude(b *strings.Builder, n *parser.Node) {
	b.WriteString(n.Fields.IncludeType)
	if len(n.Fields.Paths) > 0 {
		b.WriteByte(' ')
		b.WriteString(strings.Join(n.Fields.Paths, " "))
	}
}
