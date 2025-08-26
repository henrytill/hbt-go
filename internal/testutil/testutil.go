package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func RunHbtAndCompare(t *testing.T, format, inputFile, expectedFile string) {
	// Get absolute path to binary
	binaryPath, err := filepath.Abs("bin/hbt")
	if err != nil {
		t.Fatalf("Failed to get binary path: %v", err)
	}

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("Binary not found at %s. Run 'make all' first.", binaryPath)
	}

	// Run hbt command
	cmd := exec.Command(binaryPath, "-t", format, inputFile)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("Command failed: %v\nStderr: %s", err, exitErr.Stderr)
		}
		t.Fatalf("Command failed: %v", err)
	}

	// Read expected output
	expected, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read expected file %s: %v", expectedFile, err)
	}

	// First, a simple and fast check to see if they are identical.
	if string(output) == string(expected) {
		return
	}

	// If they are not identical, generate a temporary file for the actual output.
	actualFile, err := os.CreateTemp(t.TempDir(), "actual_*.out")
	if err != nil {
		t.Fatalf("Failed to create temp file for actual output: %v", err)
	}
	defer actualFile.Close() // Ensure the file is closed.

	if _, err := actualFile.Write(output); err != nil {
		t.Fatalf("Failed to write actual output to temp file: %v", err)
	}

	// Use the `diff` command to generate a unified diff.
	diffCmd := exec.Command("diff", "-u", expectedFile, actualFile.Name())
	diffOutput, err := diffCmd.CombinedOutput()

	// The `diff` command exits with 1 if files differ. We expect this.
	// Any other error is unexpected.
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			t.Fatalf("Failed to run diff command: %v", err)
		}
	}

	t.Logf("\n%s", string(diffOutput))

	t.Fail()
}
