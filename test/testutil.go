package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const updateExpectedEnvVar = "HBT_UPDATE_EXPECTED"

// shouldUpdateExpected checks if the environment variable specified by updateExpectedEnvVar is set to a truthy value.
func shouldUpdateExpected() bool {
	value := strings.TrimSpace(os.Getenv(updateExpectedEnvVar))
	if value == "" {
		return false
	}

	b, err := strconv.ParseBool(value)
	return err == nil && b
}

// Runs an executable with given arguments and compares the output to an expected file using diff.
func RunExecutableAndCompare(
	t *testing.T,
	executablePath string,
	args []string,
	expectedFile string,
) {
	// Check if executable exists
	if _, err := os.Stat(executablePath); os.IsNotExist(err) {
		t.Fatalf("Executable not found at %s", executablePath)
	}

	// Run command
	cmd := exec.Command(executablePath, args...)
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

	// Check if we should promote the actual output to replace the expected file
	if shouldUpdateExpected() {
		if err := os.WriteFile(expectedFile, output, 0644); err != nil {
			t.Fatalf("Failed to update expected file %s: %v", expectedFile, err)
		}
		t.Logf("Updated expected file: %s", expectedFile)
		return
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

func RunHbtAndCompare(t *testing.T, format, inputFile, expectedFile string) {
	// Get binary path - check environment variable first, fallback to relative path
	var binaryPath string
	var err error

	if envPath := os.Getenv("HBT_BINARY_PATH"); envPath != "" {
		binaryPath = envPath
	} else {
		// Fallback to relative path for local development
		binaryPath, err = filepath.Abs("../bin/hbt")
		if err != nil {
			t.Fatalf("Failed to get binary path: %v", err)
		}
	}

	// Use the general function with hbt-specific arguments
	RunExecutableAndCompare(t, binaryPath, []string{"-t", format, inputFile}, expectedFile)
}
