package pinboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient returns a client pointed at a test server running the given
// handler, plus a cleanup-registered server.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := NewClient(TokenAuth{Username: "test", Token: "token123"})
	client.baseURL = server.URL
	return client
}

func jsonHandler(t *testing.T, wantPath, body string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if wantPath != "" && !strings.HasSuffix(r.URL.Path, wantPath) {
			t.Errorf("expected path ending in %s, got %s", wantPath, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}
}

func TestGetPosts(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("dt") != "2021-01-01" {
			t.Errorf("expected dt=2021-01-01, got %q", query.Get("dt"))
		}
		if query.Get("url") != "https://example.com" {
			t.Errorf("expected url param, got %q", query.Get("url"))
		}
		if query.Get("meta") != "yes" {
			t.Errorf("expected meta=yes, got %q", query.Get("meta"))
		}
		w.Write([]byte(`{"posts": [{"href": "https://example.com", "description": "Ex"}]}`))
	})

	posts, err := client.GetPosts(context.Background(), []string{"tag1"}, "2021-01-01", "https://example.com", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 || posts[0].Href != "https://example.com" {
		t.Errorf("unexpected posts: %+v", posts)
	}
}

func TestDeletePost(t *testing.T) {
	t.Run("done", func(t *testing.T) {
		client := newTestClient(t, jsonHandler(t, "/posts/delete", `{"result_code": "done"}`))
		if err := client.DeletePost(context.Background(), "https://example.com"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("failure result", func(t *testing.T) {
		client := newTestClient(t, jsonHandler(t, "/posts/delete", `{"result_code": "item not found"}`))
		err := client.DeletePost(context.Background(), "https://example.com")
		if err == nil || !strings.Contains(err.Error(), "item not found") {
			t.Errorf("expected item not found error, got %v", err)
		}
	})
}

func TestGetPostsDates(t *testing.T) {
	client := newTestClient(t, jsonHandler(t, "/posts/dates", `{"dates": {"2021-01-01": 3, "2021-01-02": 1}}`))

	dates, err := client.GetPostsDates(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dates["2021-01-01"] != 3 || dates["2021-01-02"] != 1 {
		t.Errorf("unexpected dates: %v", dates)
	}
}

func TestGetUpdate(t *testing.T) {
	t.Run("valid time", func(t *testing.T) {
		client := newTestClient(t, jsonHandler(t, "/posts/update", `{"update_time": "2021-06-01T12:00:00Z"}`))

		got, err := client.GetUpdate(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := time.Date(2021, 6, 1, 12, 0, 0, 0, time.UTC)
		if !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("malformed time", func(t *testing.T) {
		client := newTestClient(t, jsonHandler(t, "/posts/update", `{"update_time": "yesterday"}`))
		if _, err := client.GetUpdate(context.Background()); err == nil {
			t.Error("expected error for malformed update time")
		}
	})
}

func TestSuggestTags(t *testing.T) {
	t.Run("with suggestions", func(t *testing.T) {
		client := newTestClient(t, jsonHandler(t, "/posts/suggest",
			`[{"popular": ["go"]}, {"recommended": ["golang", "web"]}]`))

		popular, recommended, err := client.SuggestTags(context.Background(), "https://example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(popular) != 1 || popular[0] != "go" {
			t.Errorf("popular = %v, want [go]", popular)
		}
		// Documents current behavior: only the first array element is
		// read, so recommended tags in the second element are dropped.
		if recommended != nil {
			t.Errorf("recommended = %v, current implementation reads only result[0]", recommended)
		}
	})

	t.Run("empty response", func(t *testing.T) {
		client := newTestClient(t, jsonHandler(t, "/posts/suggest", `[]`))

		popular, recommended, err := client.SuggestTags(context.Background(), "https://example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if popular == nil || recommended == nil {
			t.Error("expected non-nil empty slices for empty response")
		}
	})
}

func TestGetTags(t *testing.T) {
	client := newTestClient(t, jsonHandler(t, "/tags/get", `{"go": 12, "web": 3}`))

	tags, err := client.GetTags(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tags["go"] != 12 || tags["web"] != 3 {
		t.Errorf("unexpected tags: %v", tags)
	}
}

func TestDeleteTag(t *testing.T) {
	t.Run("done", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("tag"); got != "obsolete" {
				t.Errorf("expected tag=obsolete, got %q", got)
			}
			w.Write([]byte(`{"result": "done"}`))
		})
		if err := client.DeleteTag(context.Background(), "obsolete"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("failure result", func(t *testing.T) {
		client := newTestClient(t, jsonHandler(t, "/tags/delete", `{"result": "something went wrong"}`))
		if err := client.DeleteTag(context.Background(), "obsolete"); err == nil {
			t.Error("expected error for non-done result")
		}
	})
}

func TestRenameTag(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("old") != "before" || query.Get("new") != "after" {
			t.Errorf("expected old=before&new=after, got %v", query)
		}
		w.Write([]byte(`{"result": "done"}`))
	})

	if err := client.RenameTag(context.Background(), "before", "after"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListNotes(t *testing.T) {
	client := newTestClient(t, jsonHandler(t, "/notes/list",
		`{"count": 1, "notes": [{"id": "n1", "title": "First"}]}`))

	notes, err := client.ListNotes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) != 1 || notes[0].ID != "n1" || notes[0].Title != "First" {
		t.Errorf("unexpected notes: %+v", notes)
	}
}

func TestGetSecret(t *testing.T) {
	client := newTestClient(t, jsonHandler(t, "/user/secret", `{"result": "sekrit"}`))

	secret, err := client.GetSecret(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secret != "sekrit" {
		t.Errorf("got %q, want sekrit", secret)
	}
}

func TestMakeRequestServerError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := client.makeRequest(context.Background(), "posts/get", nil)
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Errorf("expected status 500 error, got %v", err)
	}
}

func TestDecodeErrorSurfaces(t *testing.T) {
	client := newTestClient(t, jsonHandler(t, "", `this is not json`))

	if _, err := client.GetTags(context.Background()); err == nil {
		t.Error("expected decode error for malformed response body")
	}
}
