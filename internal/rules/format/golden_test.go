package format_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/donaldgifford/makefmt/internal/config"
	"github.com/donaldgifford/makefmt/internal/formatter"
	"github.com/donaldgifford/makefmt/internal/parser"
	"github.com/donaldgifford/makefmt/internal/rules"
	_ "github.com/donaldgifford/makefmt/internal/rules" // Register rules via init().
	"github.com/donaldgifford/makefmt/internal/testutil"
)

func TestGoldenFiles(t *testing.T) {
	cfg := config.DefaultConfig()

	formatFn := func(input string) string {
		nodes := parser.Parse(input)
		formatted := formatter.Run(nodes, &cfg.Formatter, rules.FormatRules())
		return formatter.Write(formatted)
	}

	_, filename, _, _ := runtime.Caller(0)
	testdataDir := filepath.Join(filepath.Dir(filename), "..", "..", "..", "testdata")

	testutil.RunGoldenDir(t, testdataDir, formatFn)
}
