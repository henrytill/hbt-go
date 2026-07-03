package pinboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetNote(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/notes/abc123" {
			t.Errorf("expected path /notes/abc123, got %q", got)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": "abc123", "title": "Test Note"}`))
	}))
	defer server.Close()

	client := NewClient(TokenAuth{Username: "test", Token: "token123"})
	client.baseURL = server.URL

	note, err := client.GetNote(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if note.ID != "abc123" || note.Title != "Test Note" {
		t.Errorf("unexpected note: %+v", note)
	}
}

func TestGetNoteEscapesID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The raw (escaped) path must keep the entire ID inside the notes/
		// segment; an unescaped slash would change the requested endpoint.
		if got := r.URL.EscapedPath(); got != "/notes/..%2Fposts%2Fupdate" {
			t.Errorf("expected escaped path /notes/..%%2Fposts%%2Fupdate, got %q", got)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": "weird"}`))
	}))
	defer server.Close()

	client := NewClient(TokenAuth{Username: "test", Token: "token123"})
	client.baseURL = server.URL

	if _, err := client.GetNote(context.Background(), "../posts/update"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetNoteEmptyID(t *testing.T) {
	client := NewClient(TokenAuth{Username: "test", Token: "token123"})

	if _, err := client.GetNote(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty note ID, got nil")
	}
}
