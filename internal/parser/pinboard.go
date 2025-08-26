package parser

import (
	"encoding/json"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/henrytill/hbt-go/internal"
)

// PinboardEntry represents a bookmark entry in Pinboard JSON format
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

// PinboardParser implements parsing for Pinboard JSON files
type PinboardParser struct{}

// NewPinboardParser creates a new Pinboard parser
func NewPinboardParser() *PinboardParser {
	return &PinboardParser{}
}

// Parse parses a Pinboard JSON file and returns a Collection
func (p *PinboardParser) Parse(r io.Reader) (*internal.Collection, error) {
	var entries []PinboardEntry

	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&entries); err != nil {
		return nil, err
	}

	var nodes []internal.Node

	// Sort entries by time to match expected output order
	sort.Slice(entries, func(i, j int) bool {
		timeI, errI := time.Parse(time.RFC3339, entries[i].Time)
		timeJ, errJ := time.Parse(time.RFC3339, entries[j].Time)
		if errI != nil || errJ != nil {
			return false
		}
		return timeI.Before(timeJ)
	})

	for i, entry := range entries {
		// Parse timestamp
		timestamp, err := time.Parse(time.RFC3339, entry.Time)
		if err != nil {
			continue
		}

		// Create entity
		entity := internal.Entity{
			URI:       entry.Href,
			CreatedAt: internal.TimeToUnix(timestamp),
			UpdatedAt: []int64{},
			Names:     make(map[string]struct{}),
			Labels:    make(map[string]struct{}),
			Shared:    entry.Shared == "yes",
			ToRead:    entry.ToRead == "yes",
			IsFeed:    false,
		}

		// Add description as name if present
		if entry.Description != "" {
			entity.Names = map[string]struct{}{entry.Description: {}}
		} else {
			entity.Names = make(map[string]struct{})
		}

		// Add extended description if present
		if entry.Extended != "" {
			entity.Extended = &entry.Extended
		}

		// Parse tags (space-separated)
		entity.Labels = make(map[string]struct{})
		if entry.Tags != "" {
			tags := strings.Fields(entry.Tags)
			for _, tag := range tags {
				entity.Labels[tag] = struct{}{}
			}
		}

		// Create node
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
