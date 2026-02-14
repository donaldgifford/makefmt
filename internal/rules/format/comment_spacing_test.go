package format

import (
	"testing"

	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/parser"
)

func TestCommentSpacingAddsSpace(t *testing.T) {
	rule := &CommentSpacing{}
	cfg := &config.DefaultConfig().Formatter // default: space_after_comment is true

	node := &parser.Node{
		Type: parser.NodeComment,
		Raw:  "#comment",
		Fields: parser.NodeFields{
			Prefix: "#",
			Text:   "comment",
		},
	}

	result := rule.Format([]*parser.Node{node}, cfg)
	if result[0].Raw != "# comment" {
		t.Errorf("want %q, got %q", "# comment", result[0].Raw)
	}
}

func TestCommentSpacingAlreadySpaced(t *testing.T) {
	rule := &CommentSpacing{}
	cfg := &config.DefaultConfig().Formatter

	node := &parser.Node{
		Type: parser.NodeComment,
		Raw:  "# comment",
		Fields: parser.NodeFields{
			Prefix: "#",
			Text:   "comment",
		},
	}

	result := rule.Format([]*parser.Node{node}, cfg)
	// Should return same pointer (no modification needed).
	if result[0] != node {
		t.Error("already-spaced comment should not be cloned")
	}
}

func TestCommentSpacingSkipsDoubleHash(t *testing.T) {
	rule := &CommentSpacing{}
	cfg := &config.DefaultConfig().Formatter

	node := &parser.Node{
		Type: parser.NodeComment,
		Raw:  "## double hash",
		Fields: parser.NodeFields{
			Prefix: "##",
			Text:   "double hash",
		},
	}

	result := rule.Format([]*parser.Node{node}, cfg)
	if result[0] != node {
		t.Error("## comment should not be modified")
	}
}

func TestCommentSpacingSkipsShebang(t *testing.T) {
	rule := &CommentSpacing{}
	cfg := &config.DefaultConfig().Formatter

	node := &parser.Node{
		Type: parser.NodeComment,
		Raw:  "#!/bin/bash",
		Fields: parser.NodeFields{
			Prefix: "#",
			Text:   "!/bin/bash",
		},
	}

	result := rule.Format([]*parser.Node{node}, cfg)
	if result[0] != node {
		t.Error("shebang should not be modified")
	}
}

func TestCommentSpacingSkipsEmpty(t *testing.T) {
	rule := &CommentSpacing{}
	cfg := &config.DefaultConfig().Formatter

	node := &parser.Node{
		Type: parser.NodeComment,
		Raw:  "#",
		Fields: parser.NodeFields{
			Prefix: "#",
		},
	}

	result := rule.Format([]*parser.Node{node}, cfg)
	if result[0] != node {
		t.Error("empty comment should not be modified")
	}
}

func TestCommentSpacingDisabled(t *testing.T) {
	rule := &CommentSpacing{}
	cfg := &config.DefaultConfig().Formatter
	cfg.SpaceAfterComment = false

	node := &parser.Node{
		Type: parser.NodeComment,
		Raw:  "#comment",
		Fields: parser.NodeFields{
			Prefix: "#",
			Text:   "comment",
		},
	}

	result := rule.Format([]*parser.Node{node}, cfg)
	if result[0].Raw != "#comment" {
		t.Error("disabled rule should not modify nodes")
	}
}
