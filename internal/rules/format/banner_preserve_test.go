package format

import (
	"testing"

	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/parser"
)

func TestBannerPreservePassthrough(t *testing.T) {
	rule := &BannerPreserve{}
	cfg := &config.DefaultConfig().Formatter

	nodes := []*parser.Node{
		{Type: parser.NodeBannerComment, Raw: "###############"},
		{Type: parser.NodeSectionHeader, Raw: "##@ Development"},
		{Type: parser.NodeComment, Raw: "# regular comment"},
	}

	result := rule.Format(nodes, cfg)

	// Banner and section header should pass through unchanged.
	if result[0].Raw != "###############" {
		t.Errorf("banner: got %q", result[0].Raw)
	}
	if result[1].Raw != "##@ Development" {
		t.Errorf("section header: got %q", result[1].Raw)
	}
	// Regular comment should also be unchanged (passthrough).
	if result[2].Raw != "# regular comment" {
		t.Errorf("comment: got %q", result[2].Raw)
	}
}

func TestBannerPreserveIdentity(t *testing.T) {
	rule := &BannerPreserve{}
	cfg := &config.DefaultConfig().Formatter

	node := &parser.Node{
		Type: parser.NodeBannerComment,
		Raw:  "# ===================================",
	}

	result := rule.Format([]*parser.Node{node}, cfg)
	if result[0] != node {
		t.Error("banner preserve should return same pointer for banner nodes")
	}
}
