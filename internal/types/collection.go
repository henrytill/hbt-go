package types

import (
	"encoding/json"
	"fmt"
	"net/url"

	"golang.org/x/mod/semver"
)

type Collection struct {
	entities []Entity
	edges    [][]uint
	urls     map[string]uint
}

func NewCollection() *Collection {
	return &Collection{
		entities: []Entity{},
		edges:    [][]uint{},
		urls:     make(map[string]uint),
	}
}

func (c *Collection) Add(entity Entity) uint {
	nodeID := uint(len(c.entities))
	c.entities = append(c.entities, entity)
	c.edges = append(c.edges, []uint{})
	c.urls[entity.URI.String()] = nodeID
	return nodeID
}

func (c *Collection) findEntity(uri *url.URL) (uint, bool) {
	if uri == nil {
		return 0, false
	}
	nodeID, exists := c.urls[uri.String()]
	return nodeID, exists
}

func (c *Collection) Upsert(entity Entity) uint {
	if nodeID, exists := c.findEntity(entity.URI); exists {
		existing := &c.entities[nodeID]
		existing.absorb(entity)
		return nodeID
	}

	return c.Add(entity)
}

func (c *Collection) AddEdges(from, to uint) {
	if from >= uint(len(c.entities)) || to >= uint(len(c.entities)) {
		return
	}

	c.edges[from] = append(c.edges[from], to)
	c.edges[to] = append(c.edges[to], from)
}

func (c *Collection) ApplyMappings(mappings map[string]string) {
	for i := range c.entities {
		entity := &c.entities[i]

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

func (c *Collection) Len() int {
	return len(c.entities)
}

func (c *Collection) Entities() []Entity {
	return c.entities
}

type Version string

const ExpectedVersion Version = "v0.1.0"
const ExpectedVersionReq = "^0.1.0"

func NewVersion(v string) (Version, error) {
	if len(v) > 0 && v[0] != 'v' {
		v = "v" + v
	}
	if !semver.IsValid(v) {
		return "", fmt.Errorf("invalid semantic version: %s", v)
	}
	return Version(v), nil
}

func (v Version) String() string {
	s := string(v)
	if len(s) > 0 && s[0] == 'v' {
		return s[1:]
	}
	return s
}

func (v Version) IsCompatible() bool {
	return semver.Major(string(v)) == semver.Major(string(ExpectedVersion))
}

type nodeRepr struct {
	ID     uint       `yaml:"id"     json:"id"`
	Entity entityRepr `yaml:"entity" json:"entity"`
	Edges  []uint     `yaml:"edges"  json:"edges"`
}

type collectionRepr struct {
	Version string     `yaml:"version" json:"version"`
	Length  uint       `yaml:"length"  json:"length"`
	Value   []nodeRepr `yaml:"value"   json:"value"`
}

func (c *Collection) toRepr() collectionRepr {
	length := uint(len(c.entities))
	value := make([]nodeRepr, length)

	for i := range length {
		value[i] = nodeRepr{
			ID:     i,
			Entity: c.entities[i].toRepr(),
			Edges:  c.edges[i],
		}
	}

	return collectionRepr{
		Version: ExpectedVersion.String(),
		Length:  length,
		Value:   value,
	}
}

func (c *Collection) fromRepr(s collectionRepr) error {
	version, err := NewVersion(s.Version)
	if err != nil {
		return fmt.Errorf("invalid version in serialized data: %w", err)
	}

	if !version.IsCompatible() {
		return fmt.Errorf(
			"incompatible version: %s, expected compatible with %s",
			version.String(),
			ExpectedVersion.String(),
		)
	}

	length := len(s.Value)
	c.entities = make([]Entity, length)
	c.edges = make([][]uint, length)
	c.urls = make(map[string]uint)

	for i, serNode := range s.Value {
		var entity Entity
		if err := entity.fromRepr(serNode.Entity); err != nil {
			return err
		}
		c.entities[i] = entity
		c.edges[i] = serNode.Edges
		c.urls[entity.URI.String()] = uint(i)
	}

	return nil
}

func (c *Collection) MarshalYAML() (any, error) {
	return c.toRepr(), nil
}

func (c *Collection) UnmarshalYAML(unmarshal func(any) error) error {
	var aux collectionRepr
	if err := unmarshal(&aux); err != nil {
		return err
	}
	return c.fromRepr(aux)
}

func (c *Collection) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.toRepr())
}

func (c *Collection) UnmarshalJSON(data []byte) error {
	var aux collectionRepr
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	return c.fromRepr(aux)
}
