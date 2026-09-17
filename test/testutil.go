package test

import (
	"os"
	"path/filepath"
	"testing"
)

func hbtBinaryPath(t *testing.T) string {
	t.Helper()

	if envPath := os.Getenv("HBT_BINARY_PATH"); envPath != "" {
		return envPath
	}
	binaryPath, err := filepath.Abs("../bin/hbt")
	if err != nil {
		t.Fatalf("Failed to get binary path: %v", err)
	}
	return binaryPath
}
