package parser

import "testing"

func FuzzParse(f *testing.F) {
	// Seed with representative Makefile constructs.
	seeds := []string{
		"# comment\n",
		"VAR := value\n",
		"VAR:=value\n",
		"VAR ?= value\n",
		"target: prereq\n\t@echo hello\n",
		".PHONY: build test\n",
		"include foo.mk\n",
		"ifeq ($(OS),Linux)\nCC := gcc\nendif\n",
		"ifdef DEBUG\nCFLAGS := -g\nelse\nCFLAGS := -O2\nendif\n",
		"###############\n##@ Development\n",
		"define MY_FUNC\n\t@echo hello\nendef\n",
		"SOURCES := \\\n\tmain.go \\\n\tutils.go\n",
		"\n",
		"",
		"log-%:\n\t@grep -h '^$$*' $(MAKEFILE_LIST)\n",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(_ *testing.T, input string) {
		// The parser should never panic on any input.
		_ = Parse(input)
	})
}
