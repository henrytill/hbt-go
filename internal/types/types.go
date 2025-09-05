package types

import (
	"encoding/json"
	"io"
	"net/url"
	"sort"
	"time"
)

type Parser interface {
	Parse(r io.Reader) (*Collection, error)
}

type Formatter interface {
	Format(w io.Writer, collection *Collection) error
}

type Name string
type Label string

type Entity struct {
	URI           *url.URL
	CreatedAt     time.Time
	UpdatedAt     []time.Time
	Names         map[Name]struct{}
	Labels        map[Label]struct{}
	Shared        bool
	ToRead        bool
	IsFeed        bool
	Extended      *string
	LastVisitedAt *time.Time
}

func (e *Entity) absorb(other Entity) {
	if other.CreatedAt.Before(e.CreatedAt) {
		e.UpdatedAt = append(e.UpdatedAt, e.CreatedAt)
		e.CreatedAt = other.CreatedAt
	} else if other.CreatedAt.After(e.CreatedAt) {
		e.UpdatedAt = append(e.UpdatedAt, other.CreatedAt)
	}

	sort.Slice(e.UpdatedAt, func(i, j int) bool {
		return e.UpdatedAt[i].Before(e.UpdatedAt[j])
	})

	if e.Names == nil {
		e.Names = make(map[Name]struct{})
	}
	if e.Labels == nil {
		e.Labels = make(map[Label]struct{})
	}
	for k := range other.Names {
		e.Names[k] = struct{}{}
	}
	for k := range other.Labels {
		e.Labels[k] = struct{}{}
	}

	e.Shared = e.Shared || other.Shared
	e.ToRead = e.ToRead || other.ToRead
	e.IsFeed = e.IsFeed || other.IsFeed

	if other.Extended != nil && *other.Extended != "" {
		e.Extended = other.Extended
	}
	if other.LastVisitedAt != nil {
		e.LastVisitedAt = other.LastVisitedAt
	}
}

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

func MapToSortedSlice[K ~string](m map[K]struct{}) []string {
	if m == nil {
		return []string{}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)
	return keys
}

func SliceToMap[K ~string](slice []string) map[K]struct{} {
	m := make(map[K]struct{})
	for _, s := range slice {
		if s != "" {
			m[K(s)] = struct{}{}
		}
	}
	return m
}

func (e Entity) toSerialized() serializedEntity {
	var uriString string
	if e.URI != nil {
		uriString = e.URI.String()
	}

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

	e.Names = SliceToMap[Name](s.Names)
	e.Labels = SliceToMap[Label](s.Labels)
	e.Shared = s.Shared
	e.ToRead = s.ToRead
	e.IsFeed = s.IsFeed
	e.Extended = s.Extended
	return nil
}

type Node struct {
	ID     uint
	Entity Entity
	Edges  []uint
}

type serializedNode struct {
	ID     uint             `yaml:"id"     json:"id"`
	Entity serializedEntity `yaml:"entity" json:"entity"`
	Edges  []uint           `yaml:"edges"  json:"edges"`
}

func (n Node) toSerialized() serializedNode {
	return serializedNode{
		ID:     n.ID,
		Entity: n.Entity.toSerialized(),
		Edges:  n.Edges,
	}
}

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

type Collection struct {
	Version string
	Length  uint
	Value   []Node
}

func NewCollection() *Collection {
	return &Collection{
		Version: "0.1.0",
		Length:  0,
		Value:   []Node{},
	}
}

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

func (c *Collection) findEntityByURI(uri *url.URL) (uint, bool) {
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

func (c *Collection) UpsertEntity(entity Entity) uint {
	if nodeID, exists := c.findEntityByURI(entity.URI); exists {
		existing := &c.Value[nodeID].Entity
		existing.absorb(entity)
		return nodeID
	}

	return c.AddEntity(entity)
}

func (c *Collection) AddEdges(from, to uint) {
	if from >= uint(len(c.Value)) || to >= uint(len(c.Value)) {
		return
	}

	c.Value[from].Edges = append(c.Value[from].Edges, to)
	c.Value[to].Edges = append(c.Value[to].Edges, from)
}

func (c *Collection) ApplyMappings(mappings map[string]string) {
	for i := range c.Value {
		entity := &c.Value[i].Entity

		newLabels := make(map[Label]struct{})

		for label := range entity.Labels {
			if newLabel, exists := mappings[string(label)]; exists {
				newLabels[Label(newLabel)] = struct{}{}
			} else {
				newLabels[label] = struct{}{}
			}
		}

		entity.Labels = newLabels
	}
}

type serializedCollection struct {
	Version string           `yaml:"version" json:"version"`
	Length  uint             `yaml:"length"  json:"length"`
	Value   []serializedNode `yaml:"value"   json:"value"`
}

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

func (c *Collection) MarshalYAML() (any, error) {
	return c.toSerialized(), nil
}

func (c *Collection) UnmarshalYAML(unmarshal func(any) error) error {
	var aux serializedCollection
	if err := unmarshal(&aux); err != nil {
		return err
	}
	return c.fromSerialized(aux)
}

func (c *Collection) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.toSerialized())
}

func (c *Collection) UnmarshalJSON(data []byte) error {
	var aux serializedCollection
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	return c.fromSerialized(aux)
}
