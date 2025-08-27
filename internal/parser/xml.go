package parser

import (
	"encoding/xml"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/henrytill/hbt-go/internal"
)

// XMLParser implements parsing for Pinboard XML bookmark files
type XMLParser struct{}

// NewXMLParser creates a new XML parser
func NewXMLParser() *XMLParser {
	return &XMLParser{}
}

// Post represents a single bookmark post in Pinboard XML format
type Post struct {
	Href        string `xml:"href,attr"`
	Time        string `xml:"time,attr"`
	Description string `xml:"description,attr"`
	Extended    string `xml:"extended,attr"`
	Tag         string `xml:"tag,attr"`
	Hash        string `xml:"hash,attr"`
	Shared      string `xml:"shared,attr"`
	ToRead      string `xml:"toread,attr"`
}

// Posts represents the root XML element containing posts
type Posts struct {
	User  string `xml:"user,attr"`
	Posts []Post `xml:"post"`
}

// Parse parses a Pinboard XML bookmark file and returns a Collection
func (p *XMLParser) Parse(r io.Reader) (*internal.Collection, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Handle empty files
	if len(content) == 0 {
		return internal.NewCollection(), nil
	}

	var posts Posts
	err = xml.Unmarshal(content, &posts)
	if err != nil {
		return nil, err
	}

	collection := internal.NewCollection()

	// Sort posts by timestamp to ensure consistent ordering
	sort.Slice(posts.Posts, func(i, j int) bool {
		timeI, errI := time.Parse(time.RFC3339, posts.Posts[i].Time)
		timeJ, errJ := time.Parse(time.RFC3339, posts.Posts[j].Time)
		if errI != nil || errJ != nil {
			return false // Keep original order if parsing fails
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

func (p *XMLParser) convertPostToEntity(post Post) (internal.Entity, error) {
	// Parse time
	createdAt, err := time.Parse(time.RFC3339, post.Time)
	if err != nil {
		return internal.Entity{}, err
	}

	// Parse URL
	parsedURL, err := url.Parse(post.Href)
	if err != nil {
		return internal.Entity{}, err
	}

	// Parse names
	names := make(map[string]struct{})
	if strings.TrimSpace(post.Description) != "" {
		names[strings.TrimSpace(post.Description)] = struct{}{}
	}

	// Parse tags/labels
	labels := make(map[string]struct{})
	if strings.TrimSpace(post.Tag) != "" {
		tags := strings.Fields(post.Tag) // Split on whitespace
		for _, tag := range tags {
			labels[tag] = struct{}{}
		}
	}

	// Parse boolean fields
	shared := post.Shared == "yes"
	toRead := post.ToRead == "yes"

	// Handle extended field
	var extended *string
	if strings.TrimSpace(post.Extended) != "" {
		ext := strings.TrimSpace(post.Extended)
		extended = &ext
	}

	entity := internal.Entity{
		URI:       parsedURL,
		CreatedAt: internal.TimeToUnix(createdAt),
		UpdatedAt: []int64{},
		Names:     names,
		Labels:    labels,
		Shared:    shared,
		ToRead:    toRead,
		IsFeed:    false,
		Extended:  extended,
	}

	return entity, nil
}
