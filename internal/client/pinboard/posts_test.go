package pinboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddPostExplicitFalseOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		if query.Get("replace") != "no" {
			t.Errorf("expected replace=no, got %q", query.Get("replace"))
		}
		if query.Get("shared") != "no" {
			t.Errorf("expected shared=no, got %q", query.Get("shared"))
		}
		if query.Get("toread") != "no" {
			t.Errorf("expected toread=no, got %q", query.Get("toread"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result_code": "done"}`))
	}))
	defer server.Close()

	client := NewClient(TokenAuth{Username: "test", Token: "token123"})
	client.baseURL = server.URL

	off := false
	opts := &AddPostOptions{
		Replace: &off,
		Shared:  &off,
		ToRead:  &off,
	}

	if err := client.AddPost(context.Background(), "https://example.com", "Test", opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddPostUnsetOptionsOmitted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		for _, key := range []string{"replace", "shared", "toread"} {
			if query.Has(key) {
				t.Errorf("expected %s to be omitted, got %q", key, query.Get(key))
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result_code": "done"}`))
	}))
	defer server.Close()

	client := NewClient(TokenAuth{Username: "test", Token: "token123"})
	client.baseURL = server.URL

	if err := client.AddPost(context.Background(), "https://example.com", "Test", &AddPostOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
