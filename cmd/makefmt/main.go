// Package main is the entry point for makefmt.
package main

import (
	"flag"
	"fmt"
	"os"

	_ "github.com/donaldgifford/makefmt/internal/rules" // Register rules via init().
	"github.com/donaldgifford/makefmt/internal/runner"
)

// Build-time variables set via ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	check := flag.Bool("check", false, "exit 1 if any file is not formatted")
	diffFlag := flag.Bool("diff", false, "print unified diff of changes")
	write := flag.Bool("w", false, "write result to file")
	configPath := flag.String("config", "", "path to config file")
	quiet := flag.Bool("q", false, "suppress informational output")
	verbose := flag.Bool("v", false, "print files as they are processed")
	showVersion := flag.Bool("version", false, "print version and exit")

	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		fmt.Printf("makefmt %s (%s) %s\n", version, commit, date)
		return
	}

	opts := &runner.Options{
		Files:      flag.Args(),
		Check:      *check,
		Diff:       *diffFlag,
		Write:      *write,
		ConfigPath: *configPath,
		Quiet:      *quiet,
		Verbose:    *verbose,
	}

	os.Exit(runner.Run(opts))
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: makefmt [flags] [files...]

Format Makefile(s). With no files, reads from stdin.

Flags:
`)
	flag.PrintDefaults()
}
