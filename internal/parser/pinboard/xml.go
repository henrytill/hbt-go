package pinboard

import (
	"encoding/xml"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/henrytill/hbt-go/internal/types"
)

type XMLParser struct{}

func NewXMLParser() *XMLParser {
	return &XMLParser{}
}

func (p *XMLParser) Parse(r io.Reader) (*types.Collection, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if len(content) == 0 {
		return types.NewCollection(), nil
	}

	var posts Posts
	err = xml.Unmarshal(content, &posts)
	if err != nil {
		return nil, err
	}

	collection := types.NewCollection()

	sort.Slice(posts.Posts, func(i, j int) bool {
		timeI, errI := time.Parse(time.RFC3339, posts.Posts[i].Time)
		timeJ, errJ := time.Parse(time.RFC3339, posts.Posts[j].Time)
		if errI != nil || errJ != nil {
			return false
		}
		return timeI.Before(timeJ)
	})

	for _, post := range posts.Posts {
		entity, err := p.convertPostToEntity(post)
		if err != nil {
			return nil, err
		}
		collection.UpsertEntity(entity)
	}

	return collection, nil
}

func (p *XMLParser) convertPostToEntity(post Post) (types.Entity, error) {
	createdAt, err := time.Parse(time.RFC3339, post.Time)
	if err != nil {
		return types.Entity{}, err
	}

	parsedURL, err := url.Parse(post.Href)
	if err != nil {
		return types.Entity{}, err
	}

	names := make(map[Name]struct{})
	if trimmedDesc := strings.TrimSpace(post.Description); trimmedDesc != "" {
		names[Name(trimmedDesc)] = struct{}{}
	}

	labels := make(map[Label]struct{})
	if trimmedTags := strings.TrimSpace(post.Tags); trimmedTags != "" {
		for tag := range strings.FieldsSeq(trimmedTags) {
			labels[Label(tag)] = struct{}{}
		}
	}

	shared := post.Shared == "yes"
	toRead := post.ToRead == "yes"

	var extended *string
	if trimmedExt := strings.TrimSpace(post.Extended); trimmedExt != "" {
		extended = &trimmedExt
	}

	entity := types.Entity{
		URI:       parsedURL,
		CreatedAt: createdAt,
		UpdatedAt: []time.Time{},
		Names:     names,
		Labels:    labels,
		Shared:    shared,
		ToRead:    toRead,
		IsFeed:    false,
		Extended:  extended,
	}

	return entity, nil
}
