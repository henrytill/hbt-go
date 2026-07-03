package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func writeMappingsFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadMappingsYAML(t *testing.T) {
	path := writeMappingsFile(t, "mappings.yaml", "old: new\nalias: canonical\n")

	got, err := LoadMappings(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := Mappings{"old": "new", "alias": "canonical"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("got[%q] = %q, want %q", k, got[k], v)
		}
	}
}

func TestLoadMappingsJSON(t *testing.T) {
	path := writeMappingsFile(t, "mappings.json", `{"old": "new"}`)

	got, err := LoadMappings(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got["old"] != "new" {
		t.Errorf("got %v, want map[old:new]", got)
	}
}

func TestLoadMappingsEmptyFile(t *testing.T) {
	path := writeMappingsFile(t, "empty.yaml", "")

	got, err := LoadMappings(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("expected non-nil empty map for empty file")
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty map", got)
	}
}

func TestLoadMappingsInvalid(t *testing.T) {
	path := writeMappingsFile(t, "bad.yaml", "not: [valid: mapping\n")

	if _, err := LoadMappings(path); err == nil {
		t.Error("expected error for file that is neither valid YAML nor JSON")
	}
}

func TestLoadMappingsWrongShape(t *testing.T) {
	// Valid YAML, but a list rather than a string-to-string map.
	path := writeMappingsFile(t, "list.yaml", "- one\n- two\n")

	if _, err := LoadMappings(path); err == nil {
		t.Error("expected error for non-map mappings file")
	}
}

func TestLoadMappingsMissingFile(t *testing.T) {
	if _, err := LoadMappings(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Error("expected error for missing file")
	}
}
