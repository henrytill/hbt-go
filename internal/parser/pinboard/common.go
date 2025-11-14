package pinboard

import (
	"sort"
	"time"

	"github.com/henrytill/hbt-go/internal/pinboard"
)

func sortPostsByTime(posts []pinboard.Post) {
	sort.Slice(posts, func(i, j int) bool {
		timeI, errI := time.Parse(time.RFC3339, posts[i].Time)
		timeJ, errJ := time.Parse(time.RFC3339, posts[j].Time)
		if errI != nil || errJ != nil {
			return false
		}
		return timeI.Before(timeJ)
	})
}
