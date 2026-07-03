package pinboard

import (
	"slices"
	"time"

	"github.com/henrytill/hbt-go/internal/pinboard"
)

// sortPostsByTime sorts posts by timestamp, oldest first. Timestamps are
// parsed once up front rather than inside the comparator, and the sort is
// stable so posts with equal or unparseable timestamps keep their input
// order. Unparseable timestamps sort as the zero time; they are rejected
// later when the posts are converted to entities.
func sortPostsByTime(posts []pinboard.Post) {
	type keyedPost struct {
		post pinboard.Post
		time time.Time
	}

	keyed := make([]keyedPost, len(posts))
	for i, p := range posts {
		t, _ := time.Parse(time.RFC3339, p.Time)
		keyed[i] = keyedPost{post: p, time: t}
	}

	slices.SortStableFunc(keyed, func(a, b keyedPost) int {
		return a.time.Compare(b.time)
	})

	for i, k := range keyed {
		posts[i] = k.post
	}
}
