package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const flagsTestInput = `[
  {"href": "https://example.com/a", "time": "2021-01-01T00:00:00Z", "description": "A", "tags": "old shared"},
  {"href": "https://example.com/b", "time": "2021-01-02T00:00:00Z", "description": "B", "tags": "keep"}
]`

func writeFlagsTestInput(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(path, []byte(flagsTestInput), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func runHbt(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	binaryPath := hbtBinaryPath(t)
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("hbt binary not found at %s (run make first): %v", binaryPath, err)
	}

	var out, errOut strings.Builder
	cmd := exec.Command(binaryPath, args...)
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("failed to run hbt: %v", err)
	}
	return out.String(), errOut.String(), exitCode
}

func TestCLIInfo(t *testing.T) {
	input := writeFlagsTestInput(t)

	stdout, stderr, exitCode := runHbt(t, "--info", input)
	if exitCode != 0 {
		t.Fatalf("exit %d, stderr: %s", exitCode, stderr)
	}
	if stdout != "Collection contains 2 entities\n" {
		t.Errorf("unexpected --info output: %q", stdout)
	}
}

func TestCLIMappings(t *testing.T) {
	input := writeFlagsTestInput(t)
	mappings := filepath.Join(t.TempDir(), "mappings.yaml")
	if err := os.WriteFile(mappings, []byte("old: new\n"), 0644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, exitCode := runHbt(t, "-t", "yaml", "--mappings", mappings, input)
	if exitCode != 0 {
		t.Fatalf("exit %d, stderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "- new") {
		t.Errorf("expected remapped label 'new' in output:\n%s", stdout)
	}
	if strings.Contains(stdout, "- old") {
		t.Errorf("label 'old' should have been remapped:\n%s", stdout)
	}
	if !strings.Contains(stdout, "- keep") {
		t.Errorf("unmapped label 'keep' should be preserved:\n%s", stdout)
	}
}

func TestCLIOutputFile(t *testing.T) {
	input := writeFlagsTestInput(t)
	outFile := filepath.Join(t.TempDir(), "out.yaml")

	stdout, stderr, exitCode := runHbt(t, "-t", "yaml", "-o", outFile, input)
	if exitCode != 0 {
		t.Fatalf("exit %d, stderr: %s", exitCode, stderr)
	}
	if stdout != "" {
		t.Errorf("expected no stdout when writing to a file, got: %q", stdout)
	}

	written, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("output file not written: %v", err)
	}

	viaStdout, _, _ := runHbt(t, "-t", "yaml", input)
	if string(written) != viaStdout {
		t.Errorf("file output differs from stdout output:\nfile:\n%s\nstdout:\n%s", written, viaStdout)
	}
}

func TestCLIOutputFormatDetectedFromOutputFile(t *testing.T) {
	input := writeFlagsTestInput(t)
	outFile := filepath.Join(t.TempDir(), "out.yaml")

	// No -t: the yaml format must be detected from the -o extension.
	_, stderr, exitCode := runHbt(t, "-o", outFile, input)
	if exitCode != 0 {
		t.Fatalf("exit %d, stderr: %s", exitCode, stderr)
	}
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("output file not written: %v", err)
	}
}

func TestCLIErrors(t *testing.T) {
	input := writeFlagsTestInput(t)

	tests := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{
			name:       "missing input file",
			args:       []string{"-t", "yaml", filepath.Join(t.TempDir(), "nope.json")},
			wantStderr: "does not exist",
		},
		{
			name:       "no output format or analysis flag",
			args:       []string{input},
			wantStderr: "output format",
		},
		{
			name:       "no input file argument",
			args:       []string{"-t", "yaml"},
			wantStderr: "input file required",
		},
		{
			name:       "nonexistent mappings file",
			args:       []string{"-t", "yaml", "--mappings", filepath.Join(t.TempDir(), "nope.yaml"), input},
			wantStderr: "mappings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, stderr, exitCode := runHbt(t, tt.args...)
			if exitCode == 0 {
				t.Fatal("expected non-zero exit")
			}
			if !strings.Contains(stderr, tt.wantStderr) {
				t.Errorf("stderr %q does not contain %q", stderr, tt.wantStderr)
			}
		})
	}
}
