package pinboard

import (
	"testing"

	"github.com/henrytill/hbt-go/internal/pinboard"
)

func hrefs(posts []pinboard.Post) []string {
	out := make([]string, len(posts))
	for i, p := range posts {
		out[i] = p.Href
	}
	return out
}

func assertOrder(t *testing.T, posts []pinboard.Post, want []string) {
	t.Helper()
	got := hrefs(posts)
	if len(got) != len(want) {
		t.Fatalf("got %d posts, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order mismatch at %d: got %v, want %v", i, got, want)
		}
	}
}

func TestSortPostsByTime(t *testing.T) {
	posts := []pinboard.Post{
		{Href: "c", Time: "2021-03-01T00:00:00Z"},
		{Href: "a", Time: "2021-01-01T00:00:00Z"},
		{Href: "b", Time: "2021-02-01T00:00:00Z"},
	}

	sortPostsByTime(posts)

	assertOrder(t, posts, []string{"a", "b", "c"})
}

func TestSortPostsByTimeStableForEqualTimestamps(t *testing.T) {
	posts := []pinboard.Post{
		{Href: "later", Time: "2021-02-01T00:00:00Z"},
		{Href: "first", Time: "2021-01-01T00:00:00Z"},
		{Href: "second", Time: "2021-01-01T00:00:00Z"},
		{Href: "third", Time: "2021-01-01T00:00:00Z"},
	}

	sortPostsByTime(posts)

	assertOrder(t, posts, []string{"first", "second", "third", "later"})
}

func TestSortPostsByTimeUnparseableTimestamps(t *testing.T) {
	posts := []pinboard.Post{
		{Href: "valid", Time: "2021-01-01T00:00:00Z"},
		{Href: "bad1", Time: "not-a-timestamp"},
		{Href: "bad2", Time: ""},
	}

	// Unparseable timestamps sort as the zero time (before any valid one)
	// and keep their input order relative to each other.
	sortPostsByTime(posts)
	assertOrder(t, posts, []string{"bad1", "bad2", "valid"})

	// Re-sorting an already-sorted slice must not shuffle it.
	sortPostsByTime(posts)
	assertOrder(t, posts, []string{"bad1", "bad2", "valid"})
}
