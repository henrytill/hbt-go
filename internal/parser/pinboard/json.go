package pinboard

import (
	"encoding/json"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/henrytill/hbt-go/internal/types"
)

type JSONParser struct{}

func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

func (p *JSONParser) Parse(r io.Reader) (*types.Collection, error) {
	var entries []Post

	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&entries); err != nil {
		return nil, err
	}

	var nodes []types.Node

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

		entity := types.Entity{
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
			for tag := range strings.FieldsSeq(entry.Tags) {
				entity.Labels[Label(tag)] = struct{}{}
			}
		}

		node := types.Node{
			ID:     uint(i),
			Entity: entity,
			Edges:  []uint{},
		}

		nodes = append(nodes, node)
	}

	collection := &types.Collection{
		Version: "0.1.0",
		Length:  uint(len(nodes)),
		Value:   nodes,
	}

	return collection, nil
}
