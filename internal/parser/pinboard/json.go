package pinboard

import (
	"encoding/json"
	"io"
	"sort"
	"time"
)

func ParseJSON(r io.Reader) ([]Post, error) {
	var posts []Post

	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&posts); err != nil {
		return nil, err
	}

	sort.Slice(posts, func(i, j int) bool {
		timeI, errI := time.Parse(time.RFC3339, posts[i].Time)
		timeJ, errJ := time.Parse(time.RFC3339, posts[j].Time)
		if errI != nil || errJ != nil {
			return false
		}
		return timeI.Before(timeJ)
	})

	return posts, nil
}
