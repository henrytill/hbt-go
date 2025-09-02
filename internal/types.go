// Package internal provides core data types and operations for the hbt bookmark manager.
// This package defines the fundamental Collection, Node, and Entity types that represent
// bookmark collections and their associated metadata and relationships.
package internal

import (
	"encoding/json"
	"net/url"
	"sort"
	"time"
)

// =============================================================================
// Core Data Types
// =============================================================================

// Entity represents a bookmarked page with associated metadata and labels
type Entity struct {
	URI           *url.URL
	CreatedAt     time.Time
	UpdatedAt     []time.Time
	Names         map[string]struct{}
	Labels        map[string]struct{}
	Shared        bool
	ToRead        bool
	IsFeed        bool
	Extended      *string
	LastVisitedAt *time.Time
}

// Node represents a bookmark node in the collection graph
type Node struct {
	ID     uint
	Entity Entity
	Edges  []uint
}

// Collection represents a bookmark collection with version and metadata
type Collection struct {
	Version string
	Length  uint
	Value   []Node
}

// =============================================================================
// Serialization Types
// =============================================================================

// serializedEntity represents the Entity struct with slice fields for serialization
type serializedEntity struct {
	URI           string   `yaml:"uri"                     json:"uri"`
	CreatedAt     int64    `yaml:"createdAt"               json:"createdAt"`
	UpdatedAt     []int64  `yaml:"updatedAt"               json:"updatedAt"`
	Names         []string `yaml:"names"                   json:"names"`
	Labels        []string `yaml:"labels"                  json:"labels"`
	Shared        bool     `yaml:"shared"                  json:"shared"`
	ToRead        bool     `yaml:"toRead"                  json:"toRead"`
	IsFeed        bool     `yaml:"isFeed"                  json:"isFeed"`
	Extended      *string  `yaml:"extended,omitempty"      json:"extended,omitempty"`
	LastVisitedAt *int64   `yaml:"lastVisitedAt,omitempty" json:"lastVisitedAt,omitempty"`
}

// serializedNode represents the Node struct with serialized entity for JSON/YAML output
type serializedNode struct {
	ID     uint             `yaml:"id"     json:"id"`
	Entity serializedEntity `yaml:"entity" json:"entity"`
	Edges  []uint           `yaml:"edges"  json:"edges"`
}

// serializedCollection represents the Collection struct with serialized fields for JSON/YAML output
type serializedCollection struct {
	Version string           `yaml:"version" json:"version"`
	Length  uint             `yaml:"length"  json:"length"`
	Value   []serializedNode `yaml:"value"   json:"value"`
}

// =============================================================================
// Constructor Functions
// =============================================================================

// NewCollection creates a new empty collection
func NewCollection() *Collection {
	return &Collection{
		Version: "0.1.0",
		Length:  0,
		Value:   []Node{},
	}
}

// =============================================================================
// Core Business Logic Methods
// =============================================================================

// AddEntity adds an entity to the collection as a new node
func (c *Collection) AddEntity(entity Entity) uint {
	nodeID := c.Length
	node := Node{
		ID:     nodeID,
		Entity: entity,
		Edges:  []uint{},
	}
	c.Value = append(c.Value, node)
	c.Length++
	return nodeID
}

// FindEntityByURI finds an existing entity by URI, returns node ID and true if found
func (c *Collection) FindEntityByURI(uri *url.URL) (uint, bool) {
	if uri == nil {
		return 0, false
	}
	for _, node := range c.Value {
		if node.Entity.URI != nil && node.Entity.URI.String() == uri.String() {
			return node.ID, true
		}
	}
	return 0, false
}

// UpsertEntity adds a new entity or merges with existing entity if URI matches
func (c *Collection) UpsertEntity(entity Entity) uint {
	if nodeID, exists := c.FindEntityByURI(entity.URI); exists {
		// Merge with existing entity
		existing := &c.Value[nodeID].Entity

		// Determine earliest createdAt and merge updatedAt (following Rust logic)
		if entity.CreatedAt.Before(existing.CreatedAt) {
			// New entity is earlier, move existing createdAt to updatedAt
			existing.UpdatedAt = append(existing.UpdatedAt, existing.CreatedAt)
			existing.CreatedAt = entity.CreatedAt
		} else if entity.CreatedAt.After(existing.CreatedAt) {
			// New entity is later, add to updatedAt
			existing.UpdatedAt = append(existing.UpdatedAt, entity.CreatedAt)
		}

		// Sort updatedAt to maintain chronological order
		sort.Slice(existing.UpdatedAt, func(i, j int) bool {
			return existing.UpdatedAt[i].Before(existing.UpdatedAt[j])
		})

		// Merge names and labels using map union operations
		if existing.Names == nil {
			existing.Names = make(map[string]struct{})
		}
		if existing.Labels == nil {
			existing.Labels = make(map[string]struct{})
		}
		for k := range entity.Names {
			existing.Names[k] = struct{}{}
		}
		for k := range entity.Labels {
			existing.Labels[k] = struct{}{}
		}

		// Merge other boolean fields (OR logic - if either is true, result is true)
		existing.Shared = existing.Shared || entity.Shared
		existing.ToRead = existing.ToRead || entity.ToRead
		existing.IsFeed = existing.IsFeed || entity.IsFeed

		// Handle extended field - prefer non-empty values
		// TODO clarify how to address merging extended
		if entity.Extended != nil && *entity.Extended != "" {
			existing.Extended = entity.Extended
		}
		if entity.LastVisitedAt != nil {
			// TODO prefer newer
			existing.LastVisitedAt = entity.LastVisitedAt
		}

		return nodeID
	}

	// Add new entity
	return c.AddEntity(entity)
}

// ApplyMappings applies label transformations to all entities in the collection
func (c *Collection) ApplyMappings(mappings map[string]string) {
	for i := range c.Value {
		entity := &c.Value[i].Entity

		// Create a new labels map for transformed labels
		newLabels := make(map[string]struct{})

		// Process existing labels
		for label := range entity.Labels {
			if newLabel, exists := mappings[label]; exists {
				// Replace with mapped label
				newLabels[newLabel] = struct{}{}
			} else {
				// Keep original label
				newLabels[label] = struct{}{}
			}
		}

		// Replace labels with transformed set
		entity.Labels = newLabels
	}
}

// =============================================================================
// Utility Functions
// =============================================================================

// MapToSortedSlice converts a string set map to a sorted slice
func MapToSortedSlice(m map[string]struct{}) []string {
	if m == nil {
		return []string{}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// SliceToMap converts a string slice to a set map
func SliceToMap(slice []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, s := range slice {
		if s != "" {
			m[s] = struct{}{}
		}
	}
	return m
}

// =============================================================================
// Serialization Helper Functions
// =============================================================================

// toSerialized converts an Entity to its serialized representation
func (e Entity) toSerialized() serializedEntity {
	var uriString string
	if e.URI != nil {
		uriString = e.URI.String()
	}

	// Convert time.Time fields to Unix timestamps for serialization
	updatedAtUnix := make([]int64, len(e.UpdatedAt))
	for i, t := range e.UpdatedAt {
		updatedAtUnix[i] = t.Unix()
	}

	var lastVisitedAtUnix *int64
	if e.LastVisitedAt != nil {
		unix := e.LastVisitedAt.Unix()
		lastVisitedAtUnix = &unix
	}

	return serializedEntity{
		URI:           uriString,
		CreatedAt:     e.CreatedAt.Unix(),
		UpdatedAt:     updatedAtUnix,
		Names:         MapToSortedSlice(e.Names),
		Labels:        MapToSortedSlice(e.Labels),
		Shared:        e.Shared,
		ToRead:        e.ToRead,
		IsFeed:        e.IsFeed,
		Extended:      e.Extended,
		LastVisitedAt: lastVisitedAtUnix,
	}
}

// fromSerialized converts a serialized representation back to Entity
func (e *Entity) fromSerialized(s serializedEntity) error {
	if s.URI != "" {
		parsedURL, err := url.Parse(s.URI)
		if err != nil {
			return err
		}
		e.URI = parsedURL
	} else {
		e.URI = nil
	}

	// Convert Unix timestamps back to time.Time
	e.CreatedAt = time.Unix(s.CreatedAt, 0)

	e.UpdatedAt = make([]time.Time, len(s.UpdatedAt))
	for i, unix := range s.UpdatedAt {
		e.UpdatedAt[i] = time.Unix(unix, 0)
	}

	if s.LastVisitedAt != nil {
		t := time.Unix(*s.LastVisitedAt, 0)
		e.LastVisitedAt = &t
	} else {
		e.LastVisitedAt = nil
	}

	e.Names = SliceToMap(s.Names)
	e.Labels = SliceToMap(s.Labels)
	e.Shared = s.Shared
	e.ToRead = s.ToRead
	e.IsFeed = s.IsFeed
	e.Extended = s.Extended
	return nil
}

// toSerialized converts a Node to its serialized representation
func (n Node) toSerialized() serializedNode {
	return serializedNode{
		ID:     n.ID,
		Entity: n.Entity.toSerialized(),
		Edges:  n.Edges,
	}
}

// fromSerialized converts a serialized representation back to Node
func (n *Node) fromSerialized(s serializedNode) error {
	var entity Entity
	if err := entity.fromSerialized(s.Entity); err != nil {
		return err
	}

	n.ID = s.ID
	n.Entity = entity
	n.Edges = s.Edges
	return nil
}

// toSerialized converts a Collection to its serialized representation
func (c *Collection) toSerialized() serializedCollection {
	value := make([]serializedNode, len(c.Value))
	for i, node := range c.Value {
		value[i] = node.toSerialized()
	}

	return serializedCollection{
		Version: c.Version,
		Length:  c.Length,
		Value:   value,
	}
}

// fromSerialized converts a serialized representation back to Collection
func (c *Collection) fromSerialized(s serializedCollection) error {
	c.Version = s.Version
	c.Length = s.Length
	c.Value = make([]Node, len(s.Value))

	for i, serNode := range s.Value {
		var node Node
		if err := node.fromSerialized(serNode); err != nil {
			return err
		}
		c.Value[i] = node
	}

	return nil
}

// =============================================================================
// Marshaling/Unmarshaling Methods
// =============================================================================

// MarshalYAML implements custom YAML marshaling for Collection
func (c *Collection) MarshalYAML() (any, error) {
	return c.toSerialized(), nil
}

// UnmarshalYAML implements custom YAML unmarshaling for Collection
func (c *Collection) UnmarshalYAML(unmarshal func(any) error) error {
	var aux serializedCollection
	if err := unmarshal(&aux); err != nil {
		return err
	}
	return c.fromSerialized(aux)
}

// MarshalJSON implements custom JSON marshaling for Collection
func (c *Collection) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.toSerialized())
}

// UnmarshalJSON implements custom JSON unmarshaling for Collection
func (c *Collection) UnmarshalJSON(data []byte) error {
	var aux serializedCollection
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	return c.fromSerialized(aux)
}
