package format

import (
	"testing"

	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/formatter"
	"github.com/donaldgifford/makefmt/internal/parser"
)

// makeAssignment builds a NodeAssignment for use in tests.
func makeAssignment(varName, assignOp, varValue string) *parser.Node {
	return &parser.Node{
		Type: parser.NodeAssignment,
		Raw:  varName + " " + assignOp + " " + varValue,
		Fields: parser.NodeFields{
			VarName:  varName,
			AssignOp: assignOp,
			VarValue: varValue,
		},
	}
}

func TestAlignAssignmentsDisabled(t *testing.T) {
	rule := &AlignAssignments{}
	cfg := config.DefaultConfig().Formatter
	cfg.AlignAssignments = false

	nodes := []*parser.Node{
		makeAssignment("A", ":=", "1"),
		makeAssignment("BB", ":=", "2"),
	}

	result := rule.Format(nodes, &cfg)

	// Must return the exact same slice — no allocation, no modification.
	if len(result) != len(nodes) {
		t.Fatalf("len: got %d, want %d", len(result), len(nodes))
	}
	for i := range nodes {
		if result[i] != nodes[i] {
			t.Errorf("node[%d]: pointer changed; disabled mode must return same slice", i)
		}
	}
}

func TestAlignAssignmentsBasicGroup(t *testing.T) {
	rule := &AlignAssignments{}

	tests := []struct {
		name    string
		nodes   []*parser.Node
		want    []string // expected writer output per node
		wantRaw []string // expected Raw field (empty means writer reconstructs)
	}{
		{
			name: "group of 3",
			// PROJECT_OWNER is 13 chars — longest; all others get padded to 13.
			nodes: []*parser.Node{
				makeAssignment("PROJECT_NAME", ":=", "makefmt"),
				makeAssignment("PROJECT_OWNER", ":=", "donaldgifford"),
				makeAssignment("DESCRIPTION", ":=", "GNU Make formatter"),
			},
			want: []string{
				"PROJECT_NAME  := makefmt\n",
				"PROJECT_OWNER := donaldgifford\n",
				"DESCRIPTION   := GNU Make formatter\n",
			},
			wantRaw: []string{"", "", ""},
		},
		{
			name: "single assignment — no padding",
			nodes: []*parser.Node{
				makeAssignment("FOO", ":=", "bar"),
			},
			want:    []string{"FOO := bar\n"},
			wantRaw: []string{"FOO := bar"},
		},
		{
			name: "already aligned — idempotent",
			// Pre-padded input: VarName stored as "A  " (padded to 3).
			// TrimRight strips it back to "A" before measuring.
			nodes: []*parser.Node{
				{
					Type:   parser.NodeAssignment,
					Raw:    "A   := 1",
					Fields: parser.NodeFields{VarName: "A  ", AssignOp: ":=", VarValue: "1"},
				},
				{
					Type:   parser.NodeAssignment,
					Raw:    "ABC := 3",
					Fields: parser.NodeFields{VarName: "ABC", AssignOp: ":=", VarValue: "3"},
				},
			},
			// Longest bare name is "ABC" (3 chars). "A" → "A  " (3 chars).
			want:    []string{"A   := 1\n", "ABC := 3\n"},
			wantRaw: []string{"", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig().Formatter
			cfg.AlignAssignments = true
			cfg.AssignmentSpacing = "space"

			result := rule.Format(tt.nodes, &cfg)

			if len(result) != len(tt.want) {
				t.Fatalf("len: got %d, want %d", len(result), len(tt.want))
			}

			for i, n := range result {
				if n.Raw != tt.wantRaw[i] {
					t.Errorf("node[%d] Raw: got %q, want %q", i, n.Raw, tt.wantRaw[i])
				}
				got := formatter.Write([]*parser.Node{n})
				if got != tt.want[i] {
					t.Errorf("node[%d] output: got %q, want %q", i, got, tt.want[i])
				}
			}
		})
	}
}

func TestAlignAssignmentsGroupBoundaries(t *testing.T) {
	rule := &AlignAssignments{}

	// Two short+long pairs, separated by a boundary node.
	// Each pair forms its own group aligned to the longest in that pair.
	buildPair := func(short, long string) (*parser.Node, *parser.Node) {
		return makeAssignment(short, ":=", "x"), makeAssignment(long, ":=", "y")
	}

	a1, a2 := buildPair("AA", "BBBB")  // group 1: max=4, "AA" → "AA  "
	a3, a4 := buildPair("C", "DDDDDD") // group 2: max=6, "C" → "C     "

	tests := []struct {
		name      string
		separator *parser.Node
	}{
		{
			name:      "blank line breaks group",
			separator: &parser.Node{Type: parser.NodeBlankLine},
		},
		{
			name: "comment breaks group",
			separator: &parser.Node{
				Type:   parser.NodeComment,
				Raw:    "# separator",
				Fields: parser.NodeFields{Prefix: "#", Text: "separator"},
			},
		},
		{
			name: "rule breaks group",
			separator: &parser.Node{
				Type:   parser.NodeRule,
				Raw:    "all:",
				Fields: parser.NodeFields{Targets: []string{"all"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig().Formatter
			cfg.AlignAssignments = true
			cfg.AssignmentSpacing = "space"

			nodes := []*parser.Node{
				a1.Clone(), a2.Clone(),
				tt.separator,
				a3.Clone(), a4.Clone(),
			}

			result := rule.Format(nodes, &cfg)

			// Group 1: "AA" padded to 4, "BBBB" stays at 4.
			out0 := formatter.Write([]*parser.Node{result[0]})
			if out0 != "AA   := x\n" {
				t.Errorf("group1[0]: got %q, want %q", out0, "AA   := x\n")
			}
			out1 := formatter.Write([]*parser.Node{result[1]})
			if out1 != "BBBB := y\n" {
				t.Errorf("group1[1]: got %q, want %q", out1, "BBBB := y\n")
			}

			// Separator node must be unchanged.
			if result[2] != tt.separator {
				t.Errorf("separator node was modified")
			}

			// Group 2: "C" padded to 6, "DDDDDD" stays at 6.
			out3 := formatter.Write([]*parser.Node{result[3]})
			if out3 != "C      := x\n" {
				t.Errorf("group2[0]: got %q, want %q", out3, "C      := x\n")
			}
			out4 := formatter.Write([]*parser.Node{result[4]})
			if out4 != "DDDDDD := y\n" {
				t.Errorf("group2[1]: got %q, want %q", out4, "DDDDDD := y\n")
			}
		})
	}
}

func TestAlignAssignmentsMixedOperators(t *testing.T) {
	rule := &AlignAssignments{}
	cfg := config.DefaultConfig().Formatter
	cfg.AlignAssignments = true
	cfg.AssignmentSpacing = "space"

	// GO is 2 chars, GO_PACKAGE is 10 chars — longest.
	// All operators start at column 10+1 = 11 (after padding + writer space).
	nodes := []*parser.Node{
		makeAssignment("GO", "?=", "go"),
		makeAssignment("GO_PACKAGE", ":=", "github.com/foo/bar"),
		makeAssignment("VERSION", "+=", "extra"),
	}

	result := rule.Format(nodes, &cfg)

	wants := []string{
		"GO         ?= go\n",
		"GO_PACKAGE := github.com/foo/bar\n",
		"VERSION    += extra\n",
	}

	for i, want := range wants {
		got := formatter.Write([]*parser.Node{result[i]})
		if got != want {
			t.Errorf("node[%d]: got %q, want %q", i, got, want)
		}
	}
}

func TestAlignAssignmentsSpacingModes(t *testing.T) {
	rule := &AlignAssignments{}

	// A (1 char) and BBB (3 chars) — max=3, A padded to "A  ".
	nodes := func() []*parser.Node {
		return []*parser.Node{
			makeAssignment("A", ":=", "1"),
			makeAssignment("BBB", ":=", "2"),
		}
	}

	tests := []struct {
		name        string
		spacingMode string
		wantOutputs []string
		wantRaws    []string
	}{
		{
			name:        "space mode",
			spacingMode: "space",
			// writer reconstructs: "A   := 1", "BBB := 2"
			wantOutputs: []string{"A   := 1\n", "BBB := 2\n"},
			wantRaws:    []string{"", ""},
		},
		{
			name:        "no_space mode",
			spacingMode: "no_space",
			// Raw set directly: "A  :=1", "BBB:=2"
			wantOutputs: []string{"A  :=1\n", "BBB:=2\n"},
			wantRaws:    []string{"A  :=1", "BBB:=2"},
		},
		{
			name:        "preserve mode",
			spacingMode: "preserve",
			// Raw cleared; writer reconstructs with spaces: "A   := 1", "BBB := 2"
			wantOutputs: []string{"A   := 1\n", "BBB := 2\n"},
			wantRaws:    []string{"", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig().Formatter
			cfg.AlignAssignments = true
			cfg.AssignmentSpacing = tt.spacingMode

			result := rule.Format(nodes(), &cfg)

			for i, n := range result {
				if n.Raw != tt.wantRaws[i] {
					t.Errorf("node[%d] Raw: got %q, want %q", i, n.Raw, tt.wantRaws[i])
				}
				got := formatter.Write([]*parser.Node{n})
				if got != tt.wantOutputs[i] {
					t.Errorf("node[%d] output: got %q, want %q", i, got, tt.wantOutputs[i])
				}
			}
		})
	}
}

func TestAlignAssignmentsEdgeCases(t *testing.T) {
	rule := &AlignAssignments{}

	t.Run("override prefix", func(t *testing.T) {
		cfg := config.DefaultConfig().Formatter
		cfg.AlignAssignments = true
		cfg.AssignmentSpacing = "space"

		// "override FOO" is 12 chars; "BAR" is 3 chars.
		// Max = 12, BAR padded to "BAR         " (12 chars).
		nodes := []*parser.Node{
			makeAssignment("override FOO", ":=", "val1"),
			makeAssignment("BAR", ":=", "val2"),
		}

		result := rule.Format(nodes, &cfg)

		want0 := "override FOO := val1\n"
		want1 := "BAR          := val2\n"

		got0 := formatter.Write([]*parser.Node{result[0]})
		got1 := formatter.Write([]*parser.Node{result[1]})

		if got0 != want0 {
			t.Errorf("override node: got %q, want %q", got0, want0)
		}
		if got1 != want1 {
			t.Errorf("plain node: got %q, want %q", got1, want1)
		}
	})

	t.Run("empty VarValue", func(t *testing.T) {
		cfg := config.DefaultConfig().Formatter
		cfg.AlignAssignments = true
		cfg.AssignmentSpacing = "space"

		// Bare assignments: "VAR :=" — no value.
		nodes := []*parser.Node{
			{
				Type:   parser.NodeAssignment,
				Raw:    "VAR :=",
				Fields: parser.NodeFields{VarName: "VAR", AssignOp: ":=", VarValue: ""},
			},
			{
				Type:   parser.NodeAssignment,
				Raw:    "LONGER :=",
				Fields: parser.NodeFields{VarName: "LONGER", AssignOp: ":=", VarValue: ""},
			},
		}

		result := rule.Format(nodes, &cfg)

		// LONGER is 6 chars; VAR padded to "VAR   ".
		// Writer emits "VAR    :=" (no trailing space — VarValue empty).
		want0 := "VAR    :=\n"
		want1 := "LONGER :=\n"

		got0 := formatter.Write([]*parser.Node{result[0]})
		got1 := formatter.Write([]*parser.Node{result[1]})

		if got0 != want0 {
			t.Errorf("VAR node: got %q, want %q", got0, want0)
		}
		if got1 != want1 {
			t.Errorf("LONGER node: got %q, want %q", got1, want1)
		}

		// Raw must be cleared so the writer handles reconstruction.
		if result[0].Raw != "" {
			t.Errorf("VAR Raw: got %q, want empty", result[0].Raw)
		}
		if result[1].Raw != "" {
			t.Errorf("LONGER Raw: got %q, want empty", result[1].Raw)
		}
	})

	t.Run("over-padded input normalizes down", func(t *testing.T) {
		cfg := config.DefaultConfig().Formatter
		cfg.AlignAssignments = true
		cfg.AssignmentSpacing = "space"

		// VarName stored with excessive trailing spaces from a prior run
		// or manual editing. TrimRight strips them before measuring.
		nodes := []*parser.Node{
			{
				Type:   parser.NodeAssignment,
				Raw:    "A          := 1",
				Fields: parser.NodeFields{VarName: "A         ", AssignOp: ":=", VarValue: "1"},
			},
			{
				Type:   parser.NodeAssignment,
				Raw:    "BB         := 2",
				Fields: parser.NodeFields{VarName: "BB        ", AssignOp: ":=", VarValue: "2"},
			},
			{
				Type:   parser.NodeAssignment,
				Raw:    "CCC        := 3",
				Fields: parser.NodeFields{VarName: "CCC       ", AssignOp: ":=", VarValue: "3"},
			},
		}

		result := rule.Format(nodes, &cfg)

		// Bare names: A(1), BB(2), CCC(3). Max=3.
		// A → "A  ", BB → "BB ", CCC → "CCC".
		wants := []string{"A   := 1\n", "BB  := 2\n", "CCC := 3\n"}
		for i, want := range wants {
			got := formatter.Write([]*parser.Node{result[i]})
			if got != want {
				t.Errorf("node[%d]: got %q, want %q", i, got, want)
			}
		}
	})

	t.Run("continuation value in group", func(t *testing.T) {
		cfg := config.DefaultConfig().Formatter
		cfg.AlignAssignments = true
		cfg.AssignmentSpacing = "space"

		// SOURCES has a space-joined multi-file VarValue (as if parsed from
		// a continuation block). Alignment uses VarName length only.
		// SOURCES(7) vs X(1) — max=7, X padded to "X      ".
		nodes := []*parser.Node{
			makeAssignment("SOURCES", ":=", "main.go utils.go handler.go"),
			makeAssignment("X", ":=", "y"),
		}

		result := rule.Format(nodes, &cfg)

		want0 := "SOURCES := main.go utils.go handler.go\n"
		want1 := "X       := y\n"

		got0 := formatter.Write([]*parser.Node{result[0]})
		got1 := formatter.Write([]*parser.Node{result[1]})

		if got0 != want0 {
			t.Errorf("SOURCES: got %q, want %q", got0, want0)
		}
		if got1 != want1 {
			t.Errorf("X: got %q, want %q", got1, want1)
		}
	})
}
