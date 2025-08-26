package internal

import (
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
		if node.Entity.URI == uri {
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
		if entity.CreatedAt < existing.CreatedAt {
			// New entity is earlier, move existing createdAt to updatedAt
			existing.UpdatedAt = append(existing.UpdatedAt, existing.CreatedAt)
			existing.CreatedAt = entity.CreatedAt
		} else if entity.CreatedAt > existing.CreatedAt {
			// New entity is later, add to updatedAt
			existing.UpdatedAt = append(existing.UpdatedAt, entity.CreatedAt)
		}
		// Sort updatedAt to maintain chronological order
		sort.Slice(existing.UpdatedAt, func(i, j int) bool {
			return existing.UpdatedAt[i] < existing.UpdatedAt[j]
		})

		// Merge names and labels (avoiding duplicates and sorting)
		existing.Names = mergeStringSlices(existing.Names, entity.Names)
		existing.Labels = mergeStringSlices(existing.Labels, entity.Labels)

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

// mergeStringSlices merges two string slices, removing duplicates and sorting
func mergeStringSlices(a, b []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	// Add all strings from both slices
	for _, s := range a {
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	// Sort alphabetically
	sort.Strings(result)
	return result
}

// Helper functions for time conversion
func TimeToUnix(t time.Time) int64 {
	return t.Unix()
}

func UnixToTime(unix int64) time.Time {
	return time.Unix(unix, 0)
}
