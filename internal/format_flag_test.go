package internal

import "testing"

func TestFormatFlagSet(t *testing.T) {
	t.Run("output flag accepts repeated valid values", func(t *testing.T) {
		f := NewOutputFormatFlag()
		if err := f.Set("html"); err != nil {
			t.Fatalf("Set(html): unexpected error: %v", err)
		}
		if err := f.Set("yaml"); err != nil {
			t.Fatalf("Set(yaml) after Set(html): unexpected error: %v", err)
		}
		if f.Format != YAML {
			t.Errorf("expected last value to win, got %v", f.Format)
		}
	})

	t.Run("input flag accepts repeated valid values", func(t *testing.T) {
		f := NewInputFormatFlag()
		if err := f.Set("html"); err != nil {
			t.Fatalf("Set(html): unexpected error: %v", err)
		}
		if err := f.Set("markdown"); err != nil {
			t.Fatalf("Set(markdown) after Set(html): unexpected error: %v", err)
		}
		if f.Format != Markdown {
			t.Errorf("expected last value to win, got %v", f.Format)
		}
	})

	t.Run("input flag rejects output-only format", func(t *testing.T) {
		f := NewInputFormatFlag()
		if err := f.Set("yaml"); err == nil {
			t.Error("expected error setting input flag to yaml")
		}
	})

	t.Run("output flag rejects input-only format", func(t *testing.T) {
		f := NewOutputFormatFlag()
		if err := f.Set("markdown"); err == nil {
			t.Error("expected error setting output flag to markdown")
		}
	})

	t.Run("rejects unknown format", func(t *testing.T) {
		f := NewInputFormatFlag()
		if err := f.Set("csv"); err == nil {
			t.Error("expected error for unknown format")
		}
	})

	t.Run("is case-insensitive", func(t *testing.T) {
		f := NewInputFormatFlag()
		if err := f.Set("JSON"); err != nil {
			t.Fatalf("Set(JSON): unexpected error: %v", err)
		}
		if f.Format != JSON {
			t.Errorf("expected JSON, got %v", f.Format)
		}
	})
}
