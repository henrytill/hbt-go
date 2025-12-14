package types

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/henrytill/hbt-go/internal/pinboard"
)

type Name string
type Label string
type Extended string

type Shared bool
type ToRead bool
type IsFeed bool

type CreatedAt time.Time

func (c CreatedAt) Unix() int64 {
	return time.Time(c).Unix()
}

func (c CreatedAt) Before(other CreatedAt) bool {
	return time.Time(c).Before(time.Time(other))
}

func (c CreatedAt) After(other CreatedAt) bool {
	return time.Time(c).After(time.Time(other))
}

type UpdatedAt time.Time

func (u UpdatedAt) Unix() int64 {
	return time.Time(u).Unix()
}

func (u UpdatedAt) Before(other UpdatedAt) bool {
	return time.Time(u).Before(time.Time(other))
}

type LastVisitedAt struct{ t *time.Time }

func NewLastVisitedAt(t time.Time) LastVisitedAt {
	return LastVisitedAt{t: &t}
}

func (l LastVisitedAt) Time() (time.Time, bool) {
	if l.t == nil {
		return time.Time{}, false
	}
	return *l.t, true
}

func (l LastVisitedAt) Concat(r LastVisitedAt) LastVisitedAt {
	if l.t == nil {
		return r
	}
	if r.t == nil {
		return l
	}
	if l.t.Before(*r.t) {
		return r
	}
	return l
}

type Entity struct {
	URI           *url.URL
	CreatedAt     CreatedAt
	UpdatedAt     []UpdatedAt
	Names         map[Name]struct{}
	Labels        map[Label]struct{}
	Shared        Shared
	ToRead        ToRead
	IsFeed        IsFeed
	Extended      []Extended
	LastVisitedAt LastVisitedAt
}

func (e *Entity) absorb(other Entity) {
	if other.CreatedAt.Before(e.CreatedAt) {
		e.UpdatedAt = append(e.UpdatedAt, UpdatedAt(e.CreatedAt))
		e.CreatedAt = other.CreatedAt
	} else if other.CreatedAt.After(e.CreatedAt) {
		e.UpdatedAt = append(e.UpdatedAt, UpdatedAt(other.CreatedAt))
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

	e.Extended = append(e.Extended, other.Extended...)

	e.LastVisitedAt = e.LastVisitedAt.Concat(other.LastVisitedAt)
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
	Extended      []string `yaml:"extended,omitempty"      json:"extended,omitempty"`
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
	if t, ok := e.LastVisitedAt.Time(); ok {
		unix := t.Unix()
		lastVisitedAtUnix = &unix
	}

	var extended []string
	if len(e.Extended) > 0 {
		extended = make([]string, len(e.Extended))
		for i, ext := range e.Extended {
			extended[i] = string(ext)
		}
	}

	return entityRepr{
		URI:           uriString,
		CreatedAt:     e.CreatedAt.Unix(),
		UpdatedAt:     updatedAtUnix,
		Names:         MapToSortedSlice(e.Names),
		Labels:        MapToSortedSlice(e.Labels),
		Shared:        bool(e.Shared),
		ToRead:        bool(e.ToRead),
		IsFeed:        bool(e.IsFeed),
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

	e.CreatedAt = CreatedAt(time.Unix(s.CreatedAt, 0))

	e.UpdatedAt = make([]UpdatedAt, len(s.UpdatedAt))
	for i, unix := range s.UpdatedAt {
		e.UpdatedAt[i] = UpdatedAt(time.Unix(unix, 0))
	}

	if s.LastVisitedAt != nil {
		e.LastVisitedAt = NewLastVisitedAt(time.Unix(*s.LastVisitedAt, 0))
	} else {
		e.LastVisitedAt = LastVisitedAt{}
	}

	e.Names = sliceToMap[Name](s.Names)
	e.Labels = sliceToMap[Label](s.Labels)
	e.Shared = Shared(s.Shared)
	e.ToRead = ToRead(s.ToRead)
	e.IsFeed = IsFeed(s.IsFeed)

	if len(s.Extended) > 0 {
		e.Extended = make([]Extended, len(s.Extended))
		for i, ext := range s.Extended {
			e.Extended[i] = Extended(ext)
		}
	} else {
		e.Extended = nil
	}

	return nil
}

func NewEntityFromPost(p pinboard.Post) (Entity, error) {
	if p.Href == "" {
		return Entity{}, fmt.Errorf("empty URL in pinboard post")
	}

	createdAt, err := time.Parse(time.RFC3339, p.Time)
	if err != nil {
		return Entity{}, err
	}

	parsedURL, err := url.Parse(p.Href)
	if err != nil {
		return Entity{}, err
	}

	names := make(map[Name]struct{})
	if trimmedDesc := strings.TrimSpace(p.Description); trimmedDesc != "" {
		names[Name(trimmedDesc)] = struct{}{}
	}

	labels := make(map[Label]struct{})
	if trimmedTags := strings.TrimSpace(p.Tags); trimmedTags != "" {
		for tag := range strings.FieldsSeq(trimmedTags) {
			labels[Label(tag)] = struct{}{}
		}
	}

	shared := Shared(p.Shared == "yes")
	toRead := ToRead(p.ToRead == "yes")

	var extended []Extended
	if trimmedExt := strings.TrimSpace(p.Extended); trimmedExt != "" {
		extended = []Extended{Extended(trimmedExt)}
	}

	entity := Entity{
		URI:       parsedURL,
		CreatedAt: CreatedAt(createdAt),
		UpdatedAt: []UpdatedAt{},
		Names:     names,
		Labels:    labels,
		Shared:    shared,
		ToRead:    toRead,
		IsFeed:    IsFeed(false),
		Extended:  extended,
	}

	return entity, nil
}
