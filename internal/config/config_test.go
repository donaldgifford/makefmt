package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Formatter defaults from DESIGN.md.
	f := cfg.Formatter
	checks := []struct {
		name string
		got  any
		want any
	}{
		{"IndentStyle", f.IndentStyle, "tab"},
		{"TabWidth", f.TabWidth, 4},
		{"MaxBlankLines", f.MaxBlankLines, 2},
		{"InsertFinalNewline", f.InsertFinalNewline, true},
		{"TrimTrailingWhitespace", f.TrimTrailingWhitespace, true},
		{"AlignAssignments", f.AlignAssignments, false},
		{"AssignmentSpacing", f.AssignmentSpacing, "space"},
		{"SortPrerequisites", f.SortPrerequisites, false},
		{"AlignBackslashContinuations", f.AlignBackslashContinuations, true},
		{"BackslashColumn", f.BackslashColumn, 79},
		{"SpaceAfterComment", f.SpaceAfterComment, true},
		{"IndentConditionals", f.IndentConditionals, true},
		{"ConditionalIndent", f.ConditionalIndent, 2},
		{"RecipePrefix", f.RecipePrefix, "preserve"},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestLoadExplicitPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.yml")

	yaml := `formatter:
  max_blank_lines: 1
  assignment_spacing: no_space
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Formatter.MaxBlankLines != 1 {
		t.Errorf("MaxBlankLines: got %d, want 1", cfg.Formatter.MaxBlankLines)
	}
	if cfg.Formatter.AssignmentSpacing != "no_space" {
		t.Errorf("AssignmentSpacing: got %q, want %q", cfg.Formatter.AssignmentSpacing, "no_space")
	}

	// Verify unspecified fields retain defaults.
	if cfg.Formatter.TabWidth != 4 {
		t.Errorf("TabWidth: got %d, want 4 (default)", cfg.Formatter.TabWidth)
	}
	if !cfg.Formatter.InsertFinalNewline {
		t.Error("InsertFinalNewline: got false, want true (default)")
	}
	if !cfg.Formatter.TrimTrailingWhitespace {
		t.Error("TrimTrailingWhitespace: got false, want true (default)")
	}
}

func TestLoadNoConfigReturnsDefaults(t *testing.T) {
	// Use an empty temp dir so no config file is discovered.
	dir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}

	want := DefaultConfig()
	if cfg.Formatter != want.Formatter {
		t.Errorf("expected default config, got %+v", cfg.Formatter)
	}
}

func TestDiscoverPriority(t *testing.T) {
	dir := t.TempDir()

	content := []byte("formatter:\n  tab_width: 4\n")

	// Create all four files; makefmt.yml (first in order) should win.
	for _, name := range []string{"makefmt.yml", "makefmt.yaml", ".makefmt.yml", ".makefmt.yaml"} {
		if err := os.WriteFile(filepath.Join(dir, name), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := Discover(dir)
	want := filepath.Join(dir, "makefmt.yml")
	if got != want {
		t.Errorf("Discover = %q, want %q", got, want)
	}

	// Remove highest-priority file; makefmt.yaml should be next.
	os.Remove(filepath.Join(dir, "makefmt.yml"))
	got = Discover(dir)
	want = filepath.Join(dir, "makefmt.yaml")
	if got != want {
		t.Errorf("after removing makefmt.yml: Discover = %q, want %q", got, want)
	}

	// Remove makefmt.yaml; .makefmt.yml should be next.
	os.Remove(filepath.Join(dir, "makefmt.yaml"))
	got = Discover(dir)
	want = filepath.Join(dir, ".makefmt.yml")
	if got != want {
		t.Errorf("after removing makefmt.yaml: Discover = %q, want %q", got, want)
	}

	// Remove .makefmt.yml; .makefmt.yaml should be last.
	os.Remove(filepath.Join(dir, ".makefmt.yml"))
	got = Discover(dir)
	want = filepath.Join(dir, ".makefmt.yaml")
	if got != want {
		t.Errorf("after removing .makefmt.yml: Discover = %q, want %q", got, want)
	}
}

func TestDiscoverNoFiles(t *testing.T) {
	dir := t.TempDir()
	got := Discover(dir)
	if got != "" {
		t.Errorf("Discover in empty dir: got %q, want empty string", got)
	}
}

func TestLoadDiscovery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "makefmt.yml")

	yaml := `formatter:
  tab_width: 8
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Formatter.TabWidth != 8 {
		t.Errorf("TabWidth: got %d, want 8", cfg.Formatter.TabWidth)
	}

	// Unspecified fields should retain defaults.
	if cfg.Formatter.MaxBlankLines != 2 {
		t.Errorf("MaxBlankLines: got %d, want 2 (default)", cfg.Formatter.MaxBlankLines)
	}
}

func TestLoadPartialYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "partial.yml")

	// Only override a single field.
	yaml := `formatter:
  space_after_comment: false
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Formatter.SpaceAfterComment {
		t.Error("SpaceAfterComment: got true, want false")
	}

	// All other fields must retain their defaults.
	def := DefaultConfig()
	if cfg.Formatter.IndentStyle != def.Formatter.IndentStyle {
		t.Errorf("IndentStyle: got %q, want %q", cfg.Formatter.IndentStyle, def.Formatter.IndentStyle)
	}
	if cfg.Formatter.TabWidth != def.Formatter.TabWidth {
		t.Errorf("TabWidth: got %d, want %d", cfg.Formatter.TabWidth, def.Formatter.TabWidth)
	}
	if cfg.Formatter.MaxBlankLines != def.Formatter.MaxBlankLines {
		t.Errorf("MaxBlankLines: got %d, want %d", cfg.Formatter.MaxBlankLines, def.Formatter.MaxBlankLines)
	}
	if cfg.Formatter.InsertFinalNewline != def.Formatter.InsertFinalNewline {
		t.Errorf("InsertFinalNewline: got %v, want %v", cfg.Formatter.InsertFinalNewline, def.Formatter.InsertFinalNewline)
	}
	if cfg.Formatter.AlignBackslashContinuations != def.Formatter.AlignBackslashContinuations {
		t.Errorf("AlignBackslashContinuations: got %v, want %v",
			cfg.Formatter.AlignBackslashContinuations, def.Formatter.AlignBackslashContinuations)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yml")

	if err := os.WriteFile(path, []byte("{{{{not valid yaml"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestLoadMissingExplicitPath(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yml")
	if err == nil {
		t.Error("expected error for missing explicit path, got nil")
	}
}

func TestLoadEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yml")

	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	// Empty file should result in all defaults.
	want := DefaultConfig()
	if cfg.Formatter != want.Formatter {
		t.Errorf("expected default config for empty file, got %+v", cfg.Formatter)
	}
}

func TestLoadLintSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lint.yml")

	yaml := `lint:
  rules:
    no-trailing-whitespace: warn
  exclude:
    - "vendor/**"
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Lint.Rules["no-trailing-whitespace"] != "warn" {
		t.Errorf("Lint.Rules: got %v, want map with no-trailing-whitespace=warn", cfg.Lint.Rules)
	}
	if len(cfg.Lint.Exclude) != 1 || cfg.Lint.Exclude[0] != "vendor/**" {
		t.Errorf("Lint.Exclude: got %v, want [vendor/**]", cfg.Lint.Exclude)
	}
}
