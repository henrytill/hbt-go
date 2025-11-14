package pinboard

import (
	"encoding/json"
	"io"
	"sort"
	"time"

	"github.com/henrytill/hbt-go/internal/pinboard"
	"github.com/henrytill/hbt-go/internal/types"
)

type JSONParser struct{}

func parseJSON(r io.Reader) ([]pinboard.Post, error) {
	var posts []pinboard.Post

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

func (p *JSONParser) Parse(r io.Reader) (*types.Collection, error) {
	posts, err := parseJSON(r)
	if err != nil {
		return nil, err
	}
	return types.NewCollectionFromPosts(posts)
}
