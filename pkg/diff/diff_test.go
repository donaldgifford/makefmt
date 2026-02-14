package diff

import (
	"strings"
	"testing"
)

func TestUnifiedIdentical(t *testing.T) {
	result := Unified("test.mk", "hello\n", "hello\n")
	if result != "" {
		t.Errorf("expected empty diff for identical inputs, got:\n%s", result)
	}
}

func TestUnifiedEmptyInputs(t *testing.T) {
	tests := []struct {
		name         string
		old, updated string
		wantDiff     bool
	}{
		{"both empty", "", "", false},
		{"old empty", "", "hello\n", true},
		{"new empty", "hello\n", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Unified("test.mk", tt.old, tt.updated)
			hasDiff := result != ""
			if hasDiff != tt.wantDiff {
				t.Errorf("wantDiff=%v, got diff=%q", tt.wantDiff, result)
			}
		})
	}
}

func TestUnifiedAddition(t *testing.T) {
	old := "line1\nline2\n"
	updated := "line1\nline2\nline3\n"

	result := Unified("test.mk", old, updated)

	if !strings.Contains(result, "--- a/test.mk") {
		t.Error("missing --- header")
	}
	if !strings.Contains(result, "+++ b/test.mk") {
		t.Error("missing +++ header")
	}
	if !strings.Contains(result, "+line3\n") {
		t.Errorf("missing addition line, got:\n%s", result)
	}
}

func TestUnifiedDeletion(t *testing.T) {
	old := "line1\nline2\nline3\n"
	updated := "line1\nline3\n"

	result := Unified("test.mk", old, updated)

	if !strings.Contains(result, "-line2\n") {
		t.Errorf("missing deletion line, got:\n%s", result)
	}
}

func TestUnifiedModification(t *testing.T) {
	old := "VAR:=val\n"
	updated := "VAR := val\n"

	result := Unified("Makefile", old, updated)

	if !strings.Contains(result, "-VAR:=val\n") {
		t.Errorf("missing old line, got:\n%s", result)
	}
	if !strings.Contains(result, "+VAR := val\n") {
		t.Errorf("missing new line, got:\n%s", result)
	}
}

func TestUnifiedHunkHeaders(t *testing.T) {
	old := "line1\nline2\nline3\n"
	updated := "line1\nchanged\nline3\n"

	result := Unified("test.mk", old, updated)

	if !strings.Contains(result, "@@") {
		t.Errorf("missing @@ hunk header, got:\n%s", result)
	}
}

func TestUnifiedLargeFile(t *testing.T) {
	oldLines := make([]string, 0, 1000)
	newLines := make([]string, 0, 1000)
	for i := range 1000 {
		oldLines = append(oldLines, "line "+string(rune('A'+i%26))+"\n")
		newLines = append(newLines, "line "+string(rune('A'+i%26))+"\n")
	}
	// Change a few lines.
	newLines[500] = "changed line 500\n"
	newLines[999] = "changed line 999\n"

	old := strings.Join(oldLines, "")
	updated := strings.Join(newLines, "")

	result := Unified("large.mk", old, updated)

	if result == "" {
		t.Error("expected non-empty diff for modified large file")
	}
	if !strings.Contains(result, "+changed line 500\n") {
		t.Error("missing change at line 500")
	}
	if !strings.Contains(result, "+changed line 999\n") {
		t.Error("missing change at line 999")
	}
}

func TestUnifiedContextLines(t *testing.T) {
	// Build a file with enough lines to see context.
	lines := make([]string, 0, 20)
	for i := range 20 {
		lines = append(lines, "line"+string(rune('A'+i))+"\n")
	}
	old := strings.Join(lines, "")

	// Change line 10 (0-indexed).
	newLines := make([]string, len(lines))
	copy(newLines, lines)
	newLines[10] = "CHANGED\n"
	updated := strings.Join(newLines, "")

	result := Unified("test.mk", old, updated)

	// Should have context lines before and after the change.
	if !strings.Contains(result, " line"+string(rune('A'+7))) {
		t.Errorf("expected context line 7 before change, got:\n%s", result)
	}
	if !strings.Contains(result, " line"+string(rune('A'+13))) {
		t.Errorf("expected context line 13 after change, got:\n%s", result)
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"one line with newline", "hello\n", 1},
		{"one line no newline", "hello", 1},
		{"two lines", "a\nb\n", 2},
		{"trailing blank", "a\n\n", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := splitLines(tt.input)
			if len(lines) != tt.want {
				t.Errorf("splitLines(%q) = %d lines, want %d: %q", tt.input, len(lines), tt.want, lines)
			}
		})
	}
}
