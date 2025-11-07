package types

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"time"

	"golang.org/x/mod/semver"
)

type Parser interface {
	Parse(r io.Reader) (*Collection, error)
}

type Formatter interface {
	Format(w io.Writer, coll *Collection) error
}

type Name string
type Label string
type Extended string

type Entity struct {
	URI           *url.URL
	CreatedAt     time.Time
	UpdatedAt     []time.Time
	Names         map[Name]struct{}
	Labels        map[Label]struct{}
	Shared        bool
	ToRead        bool
	IsFeed        bool
	Extended      *Extended
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

type entityRepr struct {
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

func sliceToMap[K ~string](slice []string) map[K]struct{} {
	m := make(map[K]struct{})
	for _, s := range slice {
		if s != "" {
			m[K(s)] = struct{}{}
		}
	}
	return m
}

func (e Entity) toRepr() entityRepr {
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

	var extended *string
	if e.Extended != nil {
		s := string(*e.Extended)
		extended = &s
	}

	return entityRepr{
		URI:           uriString,
		CreatedAt:     e.CreatedAt.Unix(),
		UpdatedAt:     updatedAtUnix,
		Names:         MapToSortedSlice(e.Names),
		Labels:        MapToSortedSlice(e.Labels),
		Shared:        e.Shared,
		ToRead:        e.ToRead,
		IsFeed:        e.IsFeed,
		Extended:      extended,
		LastVisitedAt: lastVisitedAtUnix,
	}
}

func (e *Entity) fromRepr(s entityRepr) error {
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

	e.Names = sliceToMap[Name](s.Names)
	e.Labels = sliceToMap[Label](s.Labels)
	e.Shared = s.Shared
	e.ToRead = s.ToRead
	e.IsFeed = s.IsFeed

	if s.Extended != nil {
		ext := Extended(*s.Extended)
		e.Extended = &ext
	} else {
		e.Extended = nil
	}

	return nil
}

type nodeRepr struct {
	ID     uint       `yaml:"id"     json:"id"`
	Entity entityRepr `yaml:"entity" json:"entity"`
	Edges  []uint     `yaml:"edges"  json:"edges"`
}

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
const ExpectedVersionReq = "^0.1.0" // conceptual - semver package doesn't have requirement matching

func NewVersion(v string) (Version, error) {
	// Add 'v' prefix if not present for semver validation
	if len(v) > 0 && v[0] != 'v' {
		v = "v" + v
	}
	if !semver.IsValid(v) {
		return "", fmt.Errorf("invalid semantic version: %s", v)
	}
	return Version(v), nil
}

func (v Version) String() string {
	// For serialization compatibility, remove the 'v' prefix if present
	s := string(v)
	if len(s) > 0 && s[0] == 'v' {
		return s[1:]
	}
	return s
}

func (v Version) IsCompatible() bool {
	// For now, just check major version compatibility (v0.x.x)
	return semver.Major(string(v)) == semver.Major(string(ExpectedVersion))
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
