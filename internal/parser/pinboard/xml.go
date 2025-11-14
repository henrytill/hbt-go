package pinboard

import (
	"encoding/xml"
	"io"
	"sort"
	"time"

	"github.com/henrytill/hbt-go/internal/pinboard"
	"github.com/henrytill/hbt-go/internal/types"
)

type XMLParser struct{}

func parseXML(r io.Reader) ([]pinboard.Post, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if len(content) == 0 {
		return []pinboard.Post{}, nil
	}

	var posts pinboard.Posts
	err = xml.Unmarshal(content, &posts)
	if err != nil {
		return nil, err
	}

	sort.Slice(posts.Posts, func(i, j int) bool {
		timeI, errI := time.Parse(time.RFC3339, posts.Posts[i].Time)
		timeJ, errJ := time.Parse(time.RFC3339, posts.Posts[j].Time)
		if errI != nil || errJ != nil {
			return false
		}
		return timeI.Before(timeJ)
	})

	return posts.Posts, nil
}

func (p *XMLParser) Parse(r io.Reader) (*types.Collection, error) {
	posts, err := parseXML(r)
	if err != nil {
		return nil, err
	}
	return types.NewCollectionFromPosts(posts)
}
