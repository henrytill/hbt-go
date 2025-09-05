package parser

import (
	"encoding/json"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/henrytill/hbt-go/internal"
)

type PinboardEntry struct {
	Href        string `json:"href"`
	Description string `json:"description"`
	Extended    string `json:"extended"`
	Meta        string `json:"meta"`
	Hash        string `json:"hash"`
	Time        string `json:"time"`
	Shared      string `json:"shared"`
	ToRead      string `json:"toread"`
	Tags        string `json:"tags"`
}

type PinboardParser struct{}

func NewPinboardParser() *PinboardParser {
	return &PinboardParser{}
}

func (p *PinboardParser) Parse(r io.Reader) (*internal.Collection, error) {
	var entries []PinboardEntry

	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&entries); err != nil {
		return nil, err
	}

	var nodes []internal.Node

	sort.Slice(entries, func(i, j int) bool {
		timeI, errI := time.Parse(time.RFC3339, entries[i].Time)
		timeJ, errJ := time.Parse(time.RFC3339, entries[j].Time)
		if errI != nil || errJ != nil {
			return false
		}
		return timeI.Before(timeJ)
	})

	for i, entry := range entries {
		timestamp, err := time.Parse(time.RFC3339, entry.Time)
		if err != nil {
			continue
		}

		parsedURL, err := url.Parse(entry.Href)
		if err != nil {
			continue
		}

		entity := internal.Entity{
			URI:       parsedURL,
			CreatedAt: timestamp,
			UpdatedAt: []time.Time{},
			Names:     make(map[Name]struct{}),
			Labels:    make(map[Label]struct{}),
			Shared:    entry.Shared == "yes",
			ToRead:    entry.ToRead == "yes",
			IsFeed:    false,
		}

		if entry.Description != "" {
			entity.Names = map[Name]struct{}{Name(entry.Description): {}}
		} else {
			entity.Names = make(map[Name]struct{})
		}

		if entry.Extended != "" {
			entity.Extended = &entry.Extended
		}

		entity.Labels = make(map[Label]struct{})
		if entry.Tags != "" {
			tags := strings.Fields(entry.Tags)
			for _, tag := range tags {
				entity.Labels[Label(tag)] = struct{}{}
			}
		}

		node := internal.Node{
			ID:     uint(i),
			Entity: entity,
			Edges:  []uint{},
		}

		nodes = append(nodes, node)
	}

	collection := &internal.Collection{
		Version: "0.1.0",
		Length:  uint(len(nodes)),
		Value:   nodes,
	}

	return collection, nil
}
