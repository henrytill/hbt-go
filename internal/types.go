package internal

import (
	"encoding/json"
	"net/url"
	"sort"
	"time"
)

// Collection represents a serialized collection of bookmarks
type Collection struct {
	Version string `yaml:"version" json:"version"`
	Length  uint   `yaml:"length" json:"length"`
	Value   []Node `yaml:"value" json:"value"`
}

// Node represents a serialized node in the collection graph
type Node struct {
	ID     uint   `yaml:"id" json:"id"`
	Entity Entity `yaml:"entity" json:"entity"`
	Edges  []uint `yaml:"edges" json:"edges"`
}

// Entity represents a page in the collection
type Entity struct {
	URI           *url.URL            `yaml:"uri" json:"uri"`
	CreatedAt     time.Time           `yaml:"createdAt" json:"createdAt"`
	UpdatedAt     []time.Time         `yaml:"updatedAt" json:"updatedAt"`
	Names         map[string]struct{} `yaml:"names" json:"names"`
	Labels        map[string]struct{} `yaml:"labels" json:"labels"`
	Shared        bool                `yaml:"shared" json:"shared"`
	ToRead        bool                `yaml:"toRead" json:"toRead"`
	IsFeed        bool                `yaml:"isFeed" json:"isFeed"`
	Extended      *string             `yaml:"extended,omitempty" json:"extended,omitempty"`
	LastVisitedAt *time.Time          `yaml:"lastVisitedAt,omitempty" json:"lastVisitedAt,omitempty"`
}

// entitySerialized represents the Entity struct with slice fields for serialization
type entitySerialized struct {
	URI           string   `yaml:"uri" json:"uri"`
	CreatedAt     int64    `yaml:"createdAt" json:"createdAt"`
	UpdatedAt     []int64  `yaml:"updatedAt" json:"updatedAt"`
	Names         []string `yaml:"names" json:"names"`
	Labels        []string `yaml:"labels" json:"labels"`
	Shared        bool     `yaml:"shared" json:"shared"`
	ToRead        bool     `yaml:"toRead" json:"toRead"`
	IsFeed        bool     `yaml:"isFeed" json:"isFeed"`
	Extended      *string  `yaml:"extended,omitempty" json:"extended,omitempty"`
	LastVisitedAt *int64   `yaml:"lastVisitedAt,omitempty" json:"lastVisitedAt,omitempty"`
}

// toSerialized converts an Entity to its serialized representation
func (e Entity) toSerialized() entitySerialized {
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

	return entitySerialized{
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

// MarshalYAML implements custom YAML marshaling for Entity
func (e Entity) MarshalYAML() (any, error) {
	return e.toSerialized(), nil
}

// fromSerialized converts a serialized representation back to Entity
func (e *Entity) fromSerialized(s entitySerialized) error {
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

	e.Names = sliceToMap(s.Names)
	e.Labels = sliceToMap(s.Labels)
	e.Shared = s.Shared
	e.ToRead = s.ToRead
	e.IsFeed = s.IsFeed
	e.Extended = s.Extended
	return nil
}

// UnmarshalYAML implements custom YAML unmarshaling for Entity
func (e *Entity) UnmarshalYAML(unmarshal func(any) error) error {
	var aux entitySerialized
	if err := unmarshal(&aux); err != nil {
		return err
	}
	return e.fromSerialized(aux)
}

// MarshalJSON implements custom JSON marshaling for Entity
func (e Entity) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.toSerialized())
}

// UnmarshalJSON implements custom JSON unmarshaling for Entity
func (e *Entity) UnmarshalJSON(data []byte) error {
	var aux entitySerialized
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	return e.fromSerialized(aux)
}

// Helper functions for map/slice conversion
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

func sliceToMap(slice []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, s := range slice {
		if s != "" {
			m[s] = struct{}{}
		}
	}
	return m
}

// NewCollection creates a new empty collection
func NewCollection() *Collection {
	return &Collection{
		Version: "0.1.0",
		Length:  0,
		Value:   []Node{},
	}
}

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
func (c *Collection) FindEntityByURI(uri string) (uint, bool) {
	for _, node := range c.Value {
		if node.Entity.URI != nil && node.Entity.URI.String() == uri {
			return node.ID, true
		}
	}
	return 0, false
}

// UpsertEntity adds a new entity or merges with existing entity if URI matches
func (c *Collection) UpsertEntity(entity Entity) uint {
	var uriString string
	if entity.URI != nil {
		uriString = entity.URI.String()
	}
	if nodeID, exists := c.FindEntityByURI(uriString); exists {
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
		if entity.Extended != nil && *entity.Extended != "" {
			existing.Extended = entity.Extended
		}
		if entity.LastVisitedAt != nil {
			existing.LastVisitedAt = entity.LastVisitedAt
		}

		return nodeID
	} else {
		// Add new entity
		return c.AddEntity(entity)
	}
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
