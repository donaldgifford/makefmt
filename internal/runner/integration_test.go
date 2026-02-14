package runner_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// binaryPath builds the makefmt binary and returns its path.
func binaryPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "makefmt")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	cmd := exec.CommandContext(t.Context(), "go", "build", "-o", bin, "../../cmd/makefmt")
	cmd.Dir = filepath.Join(projectRoot(t), "internal", "runner")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build binary: %v\n%s", err, out)
	}
	return bin
}

func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

func TestIntegrationStdinFormat(t *testing.T) {
	bin := binaryPath(t)

	cmd := exec.CommandContext(t.Context(), bin)
	cmd.Stdin = strings.NewReader("VAR:=val\n")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "VAR := val\n" {
		t.Errorf("stdin format: got %q, want %q", string(out), "VAR := val\n")
	}
}

func TestIntegrationCheckFormatted(t *testing.T) {
	bin := binaryPath(t)

	cmd := exec.CommandContext(t.Context(), bin, "-check")
	cmd.Stdin = strings.NewReader("VAR := val\n")
	err := cmd.Run()
	if err != nil {
		t.Errorf("check formatted: expected exit 0, got %v", err)
	}
}

func TestIntegrationCheckUnformatted(t *testing.T) {
	bin := binaryPath(t)

	cmd := exec.CommandContext(t.Context(), bin, "-check")
	cmd.Stdin = strings.NewReader("VAR:=val\n")
	err := cmd.Run()
	if err == nil {
		t.Error("check unformatted: expected exit 1, got 0")
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() != 1 {
			t.Errorf("check unformatted: expected exit 1, got %d", exitErr.ExitCode())
		}
	}
}

func TestIntegrationDiff(t *testing.T) {
	bin := binaryPath(t)

	cmd := exec.CommandContext(t.Context(), bin, "-diff")
	cmd.Stdin = strings.NewReader("VAR:=val\n")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("diff with changes: expected exit 1, got 0")
	}

	output := string(out)
	if !strings.Contains(output, "-VAR:=val") {
		t.Errorf("diff missing old line: %s", output)
	}
	if !strings.Contains(output, "+VAR := val") {
		t.Errorf("diff missing new line: %s", output)
	}
}

func TestIntegrationWrite(t *testing.T) {
	bin := binaryPath(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mk")

	if err := os.WriteFile(path, []byte("VAR:=val\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.CommandContext(t.Context(), bin, path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("write: %v\n%s", err, out)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "VAR := val\n" {
		t.Errorf("file after write: got %q", string(data))
	}
}

func TestIntegrationVersion(t *testing.T) {
	bin := binaryPath(t)

	cmd := exec.CommandContext(t.Context(), bin, "-version")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if !strings.HasPrefix(string(out), "makefmt ") {
		t.Errorf("version: got %q", string(out))
	}
}

func TestIntegrationMissingFile(t *testing.T) {
	bin := binaryPath(t)

	cmd := exec.CommandContext(t.Context(), bin, "/nonexistent/file.mk")
	err := cmd.Run()
	if err == nil {
		t.Error("missing file: expected exit 2, got 0")
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() != 2 {
			t.Errorf("missing file: expected exit 2, got %d", exitErr.ExitCode())
		}
	}
}

func TestIntegrationExplicitConfig(t *testing.T) {
	bin := binaryPath(t)
	dir := t.TempDir()

	// Create config that uses no_space mode.
	configPath := filepath.Join(dir, "custom.yml")
	cfg := "formatter:\n  assignment_spacing: no_space\n"
	if err := os.WriteFile(configPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.CommandContext(t.Context(), bin, "-config", configPath)
	cmd.Stdin = strings.NewReader("VAR := val\n")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if string(out) != "VAR:=val\n" {
		t.Errorf("config no_space: got %q, want %q", string(out), "VAR:=val\n")
	}
}

func TestIntegrationMultipleFiles(t *testing.T) {
	bin := binaryPath(t)
	dir := t.TempDir()

	good := filepath.Join(dir, "good.mk")
	bad := filepath.Join(dir, "bad.mk")
	if err := os.WriteFile(good, []byte("VAR := val\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("VAR:=val\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.CommandContext(t.Context(), bin, "-check", good, bad)
	err := cmd.Run()
	if err == nil {
		t.Error("check with mixed files: expected exit 1")
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() != 1 {
			t.Errorf("check mixed: expected exit 1, got %d", exitErr.ExitCode())
		}
	}
}
