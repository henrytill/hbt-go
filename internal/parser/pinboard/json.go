package pinboard

import (
	"encoding/json"
	"io"
	"sort"
	"time"

	"github.com/henrytill/hbt-go/internal/types"
)

type PinboardJSONParser struct{}

func NewPinboardJSONParser() *PinboardJSONParser {
	return &PinboardJSONParser{}
}

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

func (p *PinboardJSONParser) Parse(r io.Reader) (*types.Collection, error) {
	posts, err := ParseJSON(r)
	if err != nil {
		return nil, err
	}
	return NewCollectionFromPosts(posts)
}
