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

func shouldUpdateExpected() bool {
	value := strings.TrimSpace(os.Getenv(updateExpectedEnvVar))
	if value == "" {
		return false
	}

	b, err := strconv.ParseBool(value)
	return err == nil && b
}

func RunExecutableAndCompare(
	t *testing.T,
	executablePath string,
	args []string,
	expectedFile string,
) {
	if _, err := os.Stat(executablePath); os.IsNotExist(err) {
		t.Fatalf("Executable not found at %s", executablePath)
	}

	cmd := exec.Command(executablePath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("Command failed: %v\nStderr: %s", err, exitErr.Stderr)
		}
		t.Fatalf("Command failed: %v", err)
	}

	expected, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read expected file %s: %v", expectedFile, err)
	}

	if string(output) == string(expected) {
		return
	}

	actualFile, err := os.CreateTemp(t.TempDir(), "actual_*.out")
	if err != nil {
		t.Fatalf("Failed to create temp file for actual output: %v", err)
	}
	defer actualFile.Close()

	if _, err := actualFile.Write(output); err != nil {
		t.Fatalf("Failed to write actual output to temp file: %v", err)
	}

	if shouldUpdateExpected() {
		if err := os.WriteFile(expectedFile, output, 0644); err != nil {
			t.Fatalf("Failed to update expected file %s: %v", expectedFile, err)
		}
		t.Logf("Updated expected file: %s", expectedFile)
		return
	}

	diffCmd := exec.Command("diff", "-u", expectedFile, actualFile.Name())
	diffOutput, err := diffCmd.CombinedOutput()

	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			t.Fatalf("Failed to run diff command: %v", err)
		}
	}

	t.Logf("\n%s", string(diffOutput))

	t.Fail()
}

func RunHbtAndCompare(t *testing.T, format, inputFile, expectedFile string) {
	var binaryPath string
	var err error

	if envPath := os.Getenv("HBT_BINARY_PATH"); envPath != "" {
		binaryPath = envPath
	} else {
		binaryPath, err = filepath.Abs("../bin/hbt")
		if err != nil {
			t.Fatalf("Failed to get binary path: %v", err)
		}
	}

	RunExecutableAndCompare(t, binaryPath, []string{"-t", format, inputFile}, expectedFile)
}
