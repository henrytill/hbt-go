package pinboard

import (
	"net/url"
	"strings"
	"time"

	"github.com/henrytill/hbt-go/internal/types"
)

type Name = types.Name
type Label = types.Label

type Post struct {
	Href        string `xml:"href,attr"        json:"href"`
	Time        string `xml:"time,attr"        json:"time"`
	Description string `xml:"description,attr" json:"description"`
	Extended    string `xml:"extended,attr"    json:"extended"`
	Tags        string `xml:"tag,attr"         json:"tags"`
	Hash        string `xml:"hash,attr"        json:"hash"`
	Shared      string `xml:"shared,attr"      json:"shared"`
	ToRead      string `xml:"toread,attr"      json:"toread"`
}

type Posts struct {
	User  string `xml:"user,attr"`
	Posts []Post `xml:"post"`
}

func (p Post) ToEntity() (types.Entity, error) {
	createdAt, err := time.Parse(time.RFC3339, p.Time)
	if err != nil {
		return types.Entity{}, err
	}

	parsedURL, err := url.Parse(p.Href)
	if err != nil {
		return types.Entity{}, err
	}

	names := make(map[Name]struct{})
	if trimmedDesc := strings.TrimSpace(p.Description); trimmedDesc != "" {
		names[Name(trimmedDesc)] = struct{}{}
	}

	labels := make(map[Label]struct{})
	if trimmedTags := strings.TrimSpace(p.Tags); trimmedTags != "" {
		for tag := range strings.FieldsSeq(trimmedTags) {
			labels[Label(tag)] = struct{}{}
		}
	}

	shared := p.Shared == "yes"
	toRead := p.ToRead == "yes"

	var extended *string
	if trimmedExt := strings.TrimSpace(p.Extended); trimmedExt != "" {
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

func NewCollectionFromPosts(posts []Post) (*types.Collection, error) {
	collection := types.NewCollection()

	for _, post := range posts {
		entity, err := post.ToEntity()
		if err != nil {
			return nil, err
		}
		collection.UpsertEntity(entity)
	}

	return collection, nil
}
