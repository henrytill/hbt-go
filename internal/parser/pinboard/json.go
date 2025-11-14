package pinboard

import (
	"encoding/json"
	"io"

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

	sortPostsByTime(posts)

	return posts, nil
}

func (p *JSONParser) Parse(r io.Reader) (*types.Collection, error) {
	posts, err := parseJSON(r)
	if err != nil {
		return nil, err
	}
	return types.NewCollectionFromPosts(posts)
}
