// Package testutil provides shared test helpers for golden file testing.
package testutil

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// Update is a flag that, when set, regenerates golden files from current output.
// Usage: go test ./... -update
var Update = flag.Bool("update", false, "update golden files")

// FormatFunc is the signature for a function that formats Makefile source.
type FormatFunc func(input string) string

// RunGolden runs a single golden file test in the given directory.
// It reads input.mk, applies formatFn, and compares against expected.mk.
func RunGolden(t *testing.T, dir string, formatFn FormatFunc) {
	t.Helper()

	inputPath := filepath.Join(dir, "input.mk")
	expectedPath := filepath.Join(dir, "expected.mk")

	inputBytes, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", inputPath, err)
	}

	actual := formatFn(string(inputBytes))

	if *Update {
		if err := os.WriteFile(expectedPath, []byte(actual), 0o644); err != nil {
			t.Fatalf("failed to update golden file %s: %v", expectedPath, err)
		}
		t.Logf("updated golden file: %s", expectedPath)
		return
	}

	expectedBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", expectedPath, err)
	}

	expected := string(expectedBytes)
	if actual != expected {
		t.Errorf("output mismatch for %s:\n--- expected\n%s\n--- actual\n%s", dir, expected, actual)
	}
}

// RunGoldenDir walks all subdirectories under testdataDir and runs
// RunGolden for each as a subtest.
func RunGoldenDir(t *testing.T, testdataDir string, formatFn FormatFunc) {
	t.Helper()

	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("failed to read testdata dir %s: %v", testdataDir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			dir := filepath.Join(testdataDir, entry.Name())
			RunGolden(t, dir, formatFn)
		})
	}
}
