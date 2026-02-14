// Package runner orchestrates the parse -> format -> output pipeline.
package runner

import (
	"fmt"
	"io"
	"os"

	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/formatter"
	"github.com/donaldgifford/makefmt/internal/parser"
	"github.com/donaldgifford/makefmt/internal/rules"
	"github.com/donaldgifford/makefmt/pkg/diff"
)

// Exit codes.
const (
	ExitOK         = 0
	ExitFormatDiff = 1
	ExitError      = 2
)

// Options configures the runner behavior.
type Options struct {
	Files      []string
	Check      bool
	Diff       bool
	Write      bool
	ConfigPath string
	Quiet      bool
	Verbose    bool
	Stdout     io.Writer
	Stderr     io.Writer
}

// Run executes the format pipeline and returns an exit code.
func Run(opts *Options) int {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}

	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		writeErr(opts.Stderr, "makefmt: %v\n", err)
		return ExitError
	}

	formatRules := rules.FormatRules()

	// stdin mode: no files given.
	if len(opts.Files) == 0 {
		return runStdin(opts, cfg, formatRules)
	}

	exitCode := ExitOK
	for _, path := range opts.Files {
		code := runFile(opts, cfg, formatRules, path)
		if code > exitCode {
			exitCode = code
		}
	}
	return exitCode
}

func runStdin(opts *Options, cfg *config.Config, formatRules []formatter.FormatRule) int {
	src, err := io.ReadAll(os.Stdin)
	if err != nil {
		writeErr(opts.Stderr, "makefmt: reading stdin: %v\n", err)
		return ExitError
	}

	input := string(src)
	output := formatInput(input, cfg, formatRules)

	if opts.Check {
		if input != output {
			return ExitFormatDiff
		}
		return ExitOK
	}

	if opts.Diff {
		d := diff.Unified("<stdin>", input, output)
		if d != "" {
			writeOut(opts.Stdout, d)
			return ExitFormatDiff
		}
		return ExitOK
	}

	writeOut(opts.Stdout, output)
	return ExitOK
}

func runFile(opts *Options, cfg *config.Config, formatRules []formatter.FormatRule, path string) int {
	src, err := os.ReadFile(path)
	if err != nil {
		writeErr(opts.Stderr, "makefmt: %v\n", err)
		return ExitError
	}

	input := string(src)
	output := formatInput(input, cfg, formatRules)

	if opts.Verbose {
		writeErr(opts.Stderr, "%s\n", path)
	}

	if opts.Check {
		if input != output {
			if !opts.Quiet {
				writeErr(opts.Stderr, "%s\n", path)
			}
			return ExitFormatDiff
		}
		return ExitOK
	}

	if opts.Diff {
		d := diff.Unified(path, input, output)
		if d != "" {
			writeOut(opts.Stdout, d)
			return ExitFormatDiff
		}
		return ExitOK
	}

	// Write mode (default for file args).
	if input == output {
		return ExitOK
	}

	if err := os.WriteFile(path, []byte(output), 0o644); err != nil {
		writeErr(opts.Stderr, "makefmt: writing %s: %v\n", path, err)
		return ExitError
	}

	return ExitOK
}

func formatInput(input string, cfg *config.Config, formatRules []formatter.FormatRule) string {
	nodes := parser.Parse(input)
	formatted := formatter.Run(nodes, &cfg.Formatter, formatRules)
	return formatter.Write(formatted)
}

// writeOut writes to stdout.
func writeOut(w io.Writer, s string) {
	fmt.Fprint(w, s)
}

// writeErr formats and writes to stderr.
func writeErr(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, format, args...)
}
