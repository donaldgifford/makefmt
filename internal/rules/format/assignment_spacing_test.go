package format

import (
	"testing"

	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/formatter"
	"github.com/donaldgifford/makefmt/internal/parser"
)

func TestAssignmentSpacingSpace(t *testing.T) {
	rule := &AssignmentSpacing{}
	cfg := &config.DefaultConfig().Formatter // default: assignment_spacing is "space"

	tests := []struct {
		name     string
		raw      string
		varName  string
		assignOp string
		varValue string
		wantRaw  string // Empty means writer reconstructs from fields.
	}{
		{"no space", "VAR:=val", "VAR", ":=", "val", ""},
		{"already spaced", "VAR := val", "VAR", ":=", "val", ""},
		{"question equals", "GO?=go", "GO", "?=", "go", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &parser.Node{
				Type: parser.NodeAssignment,
				Raw:  tt.raw,
				Fields: parser.NodeFields{
					VarName:  tt.varName,
					AssignOp: tt.assignOp,
					VarValue: tt.varValue,
				},
			}

			result := rule.Format([]*parser.Node{node}, cfg)
			n := result[0]

			if n.Raw != tt.wantRaw {
				t.Errorf("Raw: want %q, got %q", tt.wantRaw, n.Raw)
			}

			// Verify the writer produces correct output.
			output := formatter.Write([]*parser.Node{n})
			expected := tt.varName + " " + tt.assignOp + " " + tt.varValue + "\n"
			if output != expected {
				t.Errorf("writer output: want %q, got %q", expected, output)
			}
		})
	}
}

func TestAssignmentSpacingNoSpace(t *testing.T) {
	rule := &AssignmentSpacing{}
	cfg := &config.DefaultConfig().Formatter
	cfg.AssignmentSpacing = "no_space"

	node := &parser.Node{
		Type: parser.NodeAssignment,
		Raw:  "VAR := val",
		Fields: parser.NodeFields{
			VarName:  "VAR",
			AssignOp: ":=",
			VarValue: "val",
		},
	}

	result := rule.Format([]*parser.Node{node}, cfg)
	n := result[0]

	output := formatter.Write([]*parser.Node{n})
	if output != "VAR:=val\n" {
		t.Errorf("want %q, got %q", "VAR:=val\n", output)
	}
}

func TestAssignmentSpacingPreserve(t *testing.T) {
	rule := &AssignmentSpacing{}
	cfg := &config.DefaultConfig().Formatter
	cfg.AssignmentSpacing = "preserve"

	node := &parser.Node{
		Type: parser.NodeAssignment,
		Raw:  "VAR:=val",
		Fields: parser.NodeFields{
			VarName:  "VAR",
			AssignOp: ":=",
			VarValue: "val",
		},
	}

	result := rule.Format([]*parser.Node{node}, cfg)
	if result[0] != node {
		t.Error("preserve mode should return same node pointer")
	}
}
