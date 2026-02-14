package format

import (
	"strings"

	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/parser"
)

// CommentSpacing ensures a space after # in single-hash comments.
type CommentSpacing struct{}

// Name returns the config key for this rule.
func (*CommentSpacing) Name() string {
	return "space_after_comment"
}

// Format normalizes spacing after # in comment nodes.
func (*CommentSpacing) Format(nodes []*parser.Node, cfg *config.FormatterConfig) []*parser.Node {
	if !cfg.SpaceAfterComment {
		return nodes
	}

	result := make([]*parser.Node, len(nodes))
	for i, n := range nodes {
		if n.Type == parser.NodeComment && shouldNormalize(n) {
			result[i] = normalizeComment(n)
		} else {
			result[i] = n
		}
	}
	return result
}

// shouldNormalize returns true if the comment should have its spacing fixed.
// Skips: ##, ##@, shebangs (#!), empty comments (# alone), banners.
func shouldNormalize(n *parser.Node) bool {
	if n.Fields.Prefix != "#" {
		return false // Skip ##, ##@, etc.
	}

	raw := strings.TrimSpace(n.Raw)

	// Skip shebangs.
	if strings.HasPrefix(raw, "#!") {
		return false
	}

	// Skip empty comments (just "#" with nothing after).
	if raw == "#" {
		return false
	}

	return true
}

// normalizeComment ensures "# text" format (space after #).
func normalizeComment(n *parser.Node) *parser.Node {
	raw := strings.TrimSpace(n.Raw)

	// Already has space after #.
	if len(raw) > 1 && raw[1] == ' ' {
		return n
	}

	// Insert space after #.
	clone := n.Clone()
	clone.Raw = "# " + raw[1:]
	clone.Fields.Text = strings.TrimSpace(raw[1:])
	return clone
}
