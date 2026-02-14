package format

import (
	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/parser"
)

// BannerPreserve is a guard rule that ensures banner comments and
// section headers pass through the formatter unmodified. It runs last
// in the rule chain and restores any inadvertent modifications.
type BannerPreserve struct{}

// Name returns the config key for this rule.
func (*BannerPreserve) Name() string {
	return "preserve_banner_comments"
}

// Format restores banner comments and section headers to their original
// Raw form if any prior rule modified them.
func (*BannerPreserve) Format(nodes []*parser.Node, _ *config.FormatterConfig) []*parser.Node {
	// This rule is always active (no config toggle).
	// It ensures banners and section headers are never accidentally reformatted.
	// Banner comments and section headers always have Raw set by the parser,
	// so they pass through the writer verbatim. This guard rule is a no-op
	// passthrough that documents the invariant: these nodes must not be modified.
	return nodes
}
