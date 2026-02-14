package rules

import (
	"github.com/donaldgifford/makefmt/internal/rules/format"
)

func init() {
	// Rules are registered in execution order per DESIGN.md.
	// Phase 3 rules (1-4):
	RegisterFormatRule(&format.TrailingWhitespace{})
	RegisterFormatRule(&format.FinalNewline{})
	RegisterFormatRule(&format.BlankLines{})
	RegisterFormatRule(&format.AssignmentSpacing{})

	// Phase 6 rules (5-8):
	RegisterFormatRule(&format.BackslashAlign{})
	RegisterFormatRule(&format.CommentSpacing{})
	RegisterFormatRule(&format.ConditionalIndent{})
	RegisterFormatRule(&format.BannerPreserve{})
}
