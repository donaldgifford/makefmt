package runner

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/donaldgifford/makefmt/internal/rules" // Register rules via init().
)

func TestRunFormatToStdout(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mk")
	if err := os.WriteFile(path, []byte("VAR:=val\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(&Options{
		Files:  []string{path},
		Diff:   true,
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != ExitFormatDiff {
		t.Errorf("exit code: got %d, want %d", code, ExitFormatDiff)
	}
	if stdout.Len() == 0 {
		t.Error("expected diff output on stdout")
	}
}

func TestRunCheck(t *testing.T) {
	dir := t.TempDir()

	// Unformatted file.
	unformatted := filepath.Join(dir, "bad.mk")
	if err := os.WriteFile(unformatted, []byte("VAR:=val\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(&Options{
		Files:  []string{unformatted},
		Check:  true,
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != ExitFormatDiff {
		t.Errorf("check unformatted: got %d, want %d", code, ExitFormatDiff)
	}

	// Formatted file.
	formatted := filepath.Join(dir, "good.mk")
	if err := os.WriteFile(formatted, []byte("VAR := val\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	code = Run(&Options{
		Files:  []string{formatted},
		Check:  true,
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != ExitOK {
		t.Errorf("check formatted: got %d, want %d", code, ExitOK)
	}
}

func TestRunDiff(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mk")
	if err := os.WriteFile(path, []byte("VAR:=val\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(&Options{
		Files:  []string{path},
		Diff:   true,
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != ExitFormatDiff {
		t.Errorf("exit code: got %d, want %d", code, ExitFormatDiff)
	}

	output := stdout.String()
	if output == "" {
		t.Error("expected non-empty diff")
	}
	// Should contain both old and new versions.
	if !bytes.Contains(stdout.Bytes(), []byte("-VAR:=val")) {
		t.Error("diff missing old line")
	}
	if !bytes.Contains(stdout.Bytes(), []byte("+VAR := val")) {
		t.Error("diff missing new line")
	}
}

func TestRunWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mk")
	if err := os.WriteFile(path, []byte("VAR:=val\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(&Options{
		Files:  []string{path},
		Write:  true,
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != ExitOK {
		t.Errorf("exit code: got %d, want %d", code, ExitOK)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "VAR := val\n" {
		t.Errorf("file content: got %q, want %q", string(data), "VAR := val\n")
	}
}

func TestRunMissingFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(&Options{
		Files:  []string{"/nonexistent/path/test.mk"},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != ExitError {
		t.Errorf("exit code: got %d, want %d", code, ExitError)
	}
}

func TestRunAlreadyFormatted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mk")
	content := "VAR := val\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(&Options{
		Files:  []string{path},
		Diff:   true,
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != ExitOK {
		t.Errorf("exit code: got %d, want %d", code, ExitOK)
	}
	if stdout.Len() != 0 {
		t.Errorf("expected no diff output, got: %s", stdout.String())
	}
}

func TestRunMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.mk")
	bad := filepath.Join(dir, "bad.mk")

	if err := os.WriteFile(good, []byte("VAR := val\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("VAR:=val\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(&Options{
		Files:  []string{good, bad},
		Check:  true,
		Stdout: &stdout,
		Stderr: &stderr,
	})

	// One file needs formatting, so exit code should be 1.
	if code != ExitFormatDiff {
		t.Errorf("exit code: got %d, want %d", code, ExitFormatDiff)
	}
}

func TestRunVerbose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mk")
	if err := os.WriteFile(path, []byte("VAR := val\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	_ = Run(&Options{
		Files:   []string{path},
		Verbose: true,
		Stdout:  &stdout,
		Stderr:  &stderr,
	})

	if !bytes.Contains(stderr.Bytes(), []byte("test.mk")) {
		t.Errorf("verbose mode should print filename to stderr, got: %s", stderr.String())
	}
}
