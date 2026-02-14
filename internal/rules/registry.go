// Package rules manages registration of format and lint rules.
package rules

import (
	"github.com/donaldgifford/makefmt/internal/formatter"
)

var formatRules []formatter.FormatRule

// RegisterFormatRule adds a formatting rule to the registry.
// Rules are applied in the order they are registered.
func RegisterFormatRule(r formatter.FormatRule) {
	formatRules = append(formatRules, r)
}

// FormatRules returns all registered formatting rules in execution order.
func FormatRules() []formatter.FormatRule {
	return formatRules
}
