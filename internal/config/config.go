// Package config defines the configuration types and defaults for makefmt.
package config

// Config is the top-level configuration.
type Config struct {
	Formatter FormatterConfig `yaml:"formatter"`
	Lint      LintConfig      `yaml:"lint"`
}

// FormatterConfig holds all formatter settings.
type FormatterConfig struct {
	IndentStyle                 string `yaml:"indent_style"`
	TabWidth                    int    `yaml:"tab_width"`
	MaxBlankLines               int    `yaml:"max_blank_lines"`
	InsertFinalNewline          bool   `yaml:"insert_final_newline"`
	TrimTrailingWhitespace      bool   `yaml:"trim_trailing_whitespace"`
	AlignAssignments            bool   `yaml:"align_assignments"`
	AssignmentSpacing           string `yaml:"assignment_spacing"`
	SortPrerequisites           bool   `yaml:"sort_prerequisites"`
	AlignBackslashContinuations bool   `yaml:"align_backslash_continuations"`
	BackslashColumn             int    `yaml:"backslash_column"`
	SpaceAfterComment           bool   `yaml:"space_after_comment"`
	IndentConditionals          bool   `yaml:"indent_conditionals"`
	ConditionalIndent           int    `yaml:"conditional_indent"`
	RecipePrefix                string `yaml:"recipe_prefix"`
}

// LintConfig holds lint rule settings (post-MVP placeholder).
type LintConfig struct {
	Rules   map[string]string `yaml:"rules"`
	Exclude []string          `yaml:"exclude"`
}

// DefaultConfig returns a Config with all default values from DESIGN.md.
func DefaultConfig() *Config {
	return &Config{
		Formatter: FormatterConfig{
			IndentStyle:                 "tab",
			TabWidth:                    4,
			MaxBlankLines:               2,
			InsertFinalNewline:          true,
			TrimTrailingWhitespace:      true,
			AlignAssignments:            true,
			AssignmentSpacing:           "space",
			SortPrerequisites:           false,
			AlignBackslashContinuations: true,
			BackslashColumn:             79,
			SpaceAfterComment:           true,
			IndentConditionals:          true,
			ConditionalIndent:           2,
			RecipePrefix:                "preserve",
		},
	}
}
