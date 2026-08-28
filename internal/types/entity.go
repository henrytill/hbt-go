package types

import (
	"fmt"
	"maps"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/henrytill/hbt-go/internal/pinboard"
)

type Name string
type Label string
type Extended string

// optBool is a tri-state bool: unset (the zero value), false, or true.
// It is the shared implementation behind Shared, ToRead, and IsFeed, which
// stay distinct types so Entity fields cannot be mixed up.
type optBool struct {
	Bool  bool
	Valid bool
}

func newOptBool(b bool) optBool {
	return optBool{Bool: b, Valid: true}
}

func (o optBool) get() (bool, bool) {
	return o.Bool, o.Valid
}

// merge combines two values: an unset side yields the other, and two set
// values OR together.
func (o optBool) merge(p optBool) optBool {
	if !o.Valid {
		return p
	}
	if !p.Valid {
		return o
	}
	return optBool{Bool: o.Bool || p.Bool, Valid: true}
}

type Shared struct{ optBool }

func NewShared(b bool) Shared { return Shared{newOptBool(b)} }

func (s Shared) Get() (bool, bool) { return s.get() }

func (s Shared) Merge(t Shared) Shared { return Shared{s.merge(t.optBool)} }

type ToRead struct{ optBool }

func NewToRead(b bool) ToRead { return ToRead{newOptBool(b)} }

func (r ToRead) Get() (bool, bool) { return r.get() }

func (r ToRead) Merge(s ToRead) ToRead { return ToRead{r.merge(s.optBool)} }

type IsFeed struct{ optBool }

func NewIsFeed(b bool) IsFeed { return IsFeed{newOptBool(b)} }

func (f IsFeed) Get() (bool, bool) { return f.get() }

func (f IsFeed) Merge(g IsFeed) IsFeed { return IsFeed{f.merge(g.optBool)} }

type CreatedAt time.Time

func (c CreatedAt) Unix() int64 {
	return time.Time(c).Unix()
}

func (c CreatedAt) Before(d CreatedAt) bool {
	return time.Time(c).Before(time.Time(d))
}

func (c CreatedAt) After(d CreatedAt) bool {
	return time.Time(c).After(time.Time(d))
}

type UpdatedAt time.Time

func (u UpdatedAt) Unix() int64 {
	return time.Time(u).Unix()
}

func (u UpdatedAt) Before(v UpdatedAt) bool {
	return time.Time(u).Before(time.Time(v))
}

type LastVisitedAt struct {
	Time  time.Time
	Valid bool
}

func NewLastVisitedAt(t time.Time) LastVisitedAt {
	return LastVisitedAt{Time: t, Valid: true}
}

func (l LastVisitedAt) Get() (time.Time, bool) {
	return l.Time, l.Valid
}

// Equal reports whether l and m denote the same instant, or are both unset.
func (l LastVisitedAt) Equal(m LastVisitedAt) bool {
	if l.Valid != m.Valid {
		return false
	}
	return !l.Valid || l.Time.Equal(m.Time)
}

func (l LastVisitedAt) Merge(m LastVisitedAt) LastVisitedAt {
	if !l.Valid {
		return m
	}
	if !m.Valid {
		return l
	}
	if l.Time.Before(m.Time) {
		return m
	}
	return l
}

type Entity struct {
	URI           *url.URL
	CreatedAt     CreatedAt
	UpdatedAt     []UpdatedAt
	Names         Set[Name]
	Labels        Set[Label]
	Shared        Shared
	ToRead        ToRead
	IsFeed        IsFeed
	Extended      Set[Extended]
	LastVisitedAt LastVisitedAt
}

// Equal reports whether e and other carry the same data. Times compare by
// instant rather than by representation, Names, Labels and Extended by set
// membership, and UpdatedAt element by element.
func (e Entity) Equal(other Entity) bool {
	if (e.URI == nil) != (other.URI == nil) {
		return false
	}
	if e.URI != nil && e.URI.String() != other.URI.String() {
		return false
	}
	if !time.Time(e.CreatedAt).Equal(time.Time(other.CreatedAt)) {
		return false
	}
	sameInstant := func(u, v UpdatedAt) bool { return time.Time(u).Equal(time.Time(v)) }
	if !slices.EqualFunc(e.UpdatedAt, other.UpdatedAt, sameInstant) {
		return false
	}
	if !maps.Equal(e.Names, other.Names) || !maps.Equal(e.Labels, other.Labels) {
		return false
	}
	if !maps.Equal(e.Extended, other.Extended) {
		return false
	}
	if e.Shared != other.Shared || e.ToRead != other.ToRead || e.IsFeed != other.IsFeed {
		return false
	}
	return e.LastVisitedAt.Equal(other.LastVisitedAt)
}

// absorb merges other into e. The two behaviors commented below are shared with
// hbt-ocaml and hbt-rs, settled in #57 and pinned by fixtures in hbt-data.
func (e *Entity) absorb(other Entity) {
	// Absorbing an identical entity is a no-op. Every field below merges by
	// union or by a comparison, so the merge would reach the same result on
	// its own; the guard just says so directly. See the bookmarks_repeated
	// fixture.
	if e.Equal(other) {
		return
	}

	// A timestamp equal to the existing CreatedAt is deliberately not recorded:
	// an "update" whose timestamp merely repeats CreatedAt carries no
	// information. See the bookmarks_same_timestamp fixture.
	if other.CreatedAt.Before(e.CreatedAt) {
		e.UpdatedAt = append(e.UpdatedAt, UpdatedAt(e.CreatedAt))
		e.CreatedAt = other.CreatedAt
	} else if other.CreatedAt.After(e.CreatedAt) {
		e.UpdatedAt = append(e.UpdatedAt, UpdatedAt(other.CreatedAt))
	}

	sort.Slice(e.UpdatedAt, func(i, j int) bool {
		return e.UpdatedAt[i].Before(e.UpdatedAt[j])
	})

	e.Names = e.Names.Merge(other.Names)
	e.Labels = e.Labels.Merge(other.Labels)
	e.Extended = e.Extended.Merge(other.Extended)

	e.Shared = e.Shared.Merge(other.Shared)
	e.ToRead = e.ToRead.Merge(other.ToRead)
	e.IsFeed = e.IsFeed.Merge(other.IsFeed)

	e.LastVisitedAt = e.LastVisitedAt.Merge(other.LastVisitedAt)
}

type entityRepr struct {
	URI           string   `yaml:"uri"                     json:"uri"`
	CreatedAt     int64    `yaml:"createdAt"               json:"createdAt"`
	UpdatedAt     []int64  `yaml:"updatedAt"               json:"updatedAt"`
	Names         []string `yaml:"names"                   json:"names"`
	Labels        []string `yaml:"labels"                  json:"labels"`
	Shared        *bool    `yaml:"shared,omitempty"        json:"shared,omitempty"`
	ToRead        *bool    `yaml:"toRead,omitempty"        json:"toRead,omitempty"`
	IsFeed        *bool    `yaml:"isFeed,omitempty"        json:"isFeed,omitempty"`
	Extended      []string `yaml:"extended,omitempty"      json:"extended,omitempty"`
	LastVisitedAt *int64   `yaml:"lastVisitedAt,omitempty" json:"lastVisitedAt,omitempty"`
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

	var lastVisitedAt *int64
	if t, ok := e.LastVisitedAt.Get(); ok {
		unix := t.Unix()
		lastVisitedAt = &unix
	}

	var shared *bool
	if s, ok := e.Shared.Get(); ok {
		shared = &s
	}

	var toRead *bool
	if tr, ok := e.ToRead.Get(); ok {
		toRead = &tr
	}

	var isFeed *bool
	if f, ok := e.IsFeed.Get(); ok {
		isFeed = &f
	}

	return entityRepr{
		URI:           uriString,
		CreatedAt:     e.CreatedAt.Unix(),
		UpdatedAt:     updatedAtUnix,
		Names:         SortedSlice(e.Names),
		Labels:        SortedSlice(e.Labels),
		Shared:        shared,
		ToRead:        toRead,
		IsFeed:        isFeed,
		Extended:      SortedSlice(e.Extended),
		LastVisitedAt: lastVisitedAt,
	}
}

func (e *Entity) fromRepr(s entityRepr) error {
	if s.URI == "" {
		return fmt.Errorf("missing uri")
	}
	parsedURL, err := url.Parse(s.URI)
	if err != nil {
		return err
	}
	e.URI = parsedURL

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

	e.Names = sliceToSet[Name](s.Names)
	e.Labels = sliceToSet[Label](s.Labels)

	if s.Shared != nil {
		e.Shared = NewShared(*s.Shared)
	} else {
		e.Shared = Shared{}
	}

	if s.ToRead != nil {
		e.ToRead = NewToRead(*s.ToRead)
	} else {
		e.ToRead = ToRead{}
	}

	if s.IsFeed != nil {
		e.IsFeed = NewIsFeed(*s.IsFeed)
	} else {
		e.IsFeed = IsFeed{}
	}

	e.Extended = sliceToSet[Extended](s.Extended)

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

	names := make(Set[Name])
	if trimmedDesc := strings.TrimSpace(p.Description); trimmedDesc != "" {
		names[Name(trimmedDesc)] = struct{}{}
	}

	labels := make(Set[Label])
	if trimmedTags := strings.TrimSpace(p.Tags); trimmedTags != "" {
		for tag := range strings.FieldsSeq(trimmedTags) {
			labels[Label(tag)] = struct{}{}
		}
	}

	var extended Set[Extended]
	if trimmedExt := strings.TrimSpace(p.Extended); trimmedExt != "" {
		extended = NewSet(Extended(trimmedExt))
	}

	entity := Entity{
		URI:       parsedURL,
		CreatedAt: CreatedAt(createdAt),
		UpdatedAt: []UpdatedAt{},
		Names:     names,
		Labels:    labels,
		Shared:    NewShared(p.Shared == "yes"),
		ToRead:    NewToRead(p.ToRead == "yes"),
		IsFeed:    NewIsFeed(false),
		Extended:  extended,
	}

	return entity, nil
}
