package types

import (
	"fmt"
	"maps"
	"net/url"
	"slices"
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

// timestamp is an instant as a Unix second count, the resolution the wire
// format carries. It is not a time.Time because == on one compares the
// monotonic reading and the *Location as well as the instant, which would let
// a Set[UpdatedAt] hold two members denoting the same moment. It is the shared
// implementation behind CreatedAt, UpdatedAt, and LastVisitedAt, which stay
// distinct types so Entity fields cannot be mixed up.
type timestamp int64

func (t timestamp) unix() int64 {
	return int64(t)
}

type CreatedAt struct{ timestamp }

func NewCreatedAt(unix int64) CreatedAt { return CreatedAt{timestamp(unix)} }

func (c CreatedAt) Unix() int64 { return c.unix() }

func (c CreatedAt) Before(d CreatedAt) bool { return c.timestamp < d.timestamp }

func (c CreatedAt) After(d CreatedAt) bool { return c.timestamp > d.timestamp }

type UpdatedAt struct{ timestamp }

func NewUpdatedAt(unix int64) UpdatedAt { return UpdatedAt{timestamp(unix)} }

func (u UpdatedAt) Unix() int64 { return u.unix() }

func sortedUnix(s Set[UpdatedAt]) []int64 {
	unix := make([]int64, 0, len(s))
	for u := range s {
		unix = append(unix, u.unix())
	}
	slices.Sort(unix)
	return unix
}

func unixToSet(unix []int64) Set[UpdatedAt] {
	s := make(Set[UpdatedAt], len(unix))
	for _, u := range unix {
		s[NewUpdatedAt(u)] = struct{}{}
	}
	return s
}

// LastVisitedAt is an optional visit instant: unset (the zero value) or a
// timestamp.
type LastVisitedAt struct {
	timestamp
	Valid bool
}

func NewLastVisitedAt(unix int64) LastVisitedAt {
	return LastVisitedAt{timestamp(unix), true}
}

// Get returns the instant as a Unix second count, and whether it is set.
func (l LastVisitedAt) Get() (int64, bool) {
	return l.unix(), l.Valid
}

// Equal reports whether l and m denote the same instant, or are both unset.
func (l LastVisitedAt) Equal(m LastVisitedAt) bool {
	if l.Valid != m.Valid {
		return false
	}
	return !l.Valid || l.timestamp == m.timestamp
}

func (l LastVisitedAt) Merge(m LastVisitedAt) LastVisitedAt {
	if !l.Valid {
		return m
	}
	if !m.Valid {
		return l
	}
	if l.timestamp < m.timestamp {
		return m
	}
	return l
}

type Entity struct {
	URI           *url.URL
	CreatedAt     CreatedAt
	UpdatedAt     Set[UpdatedAt]
	Names         Set[Name]
	Labels        Set[Label]
	Shared        Shared
	ToRead        ToRead
	IsFeed        IsFeed
	Extended      Set[Extended]
	LastVisitedAt LastVisitedAt
}

// Equal reports whether e and other carry the same data. UpdatedAt, Names,
// Labels and Extended compare by set membership.
func (e Entity) Equal(other Entity) bool {
	if (e.URI == nil) != (other.URI == nil) {
		return false
	}
	if e.URI != nil && e.URI.String() != other.URI.String() {
		return false
	}
	if e.CreatedAt != other.CreatedAt {
		return false
	}
	if !maps.Equal(e.UpdatedAt, other.UpdatedAt) {
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

// EarliestUpdate returns the earliest recorded update instant, and whether
// there is one.
func (e Entity) EarliestUpdate() (UpdatedAt, bool) {
	var earliest UpdatedAt
	found := false
	for u := range e.UpdatedAt {
		if !found || u.timestamp < earliest.timestamp {
			earliest, found = u, true
		}
	}
	return earliest, found
}

// absorb merges other into e. The two behaviors commented below are shared with
// hbt-ocaml and hbt-rs, settled in #57 and pinned by fixtures in hbt-data.
func (e *Entity) absorb(other Entity) {
	// Absorbing an identical entity is a no-op, which the guard states
	// directly rather than leaving to the merge below, where every field
	// merges by union or by a comparison. Fixture: bookmarks_repeated.
	if e.Equal(other) {
		return
	}

	// A timestamp equal to the existing CreatedAt is deliberately not recorded:
	// an "update" whose timestamp merely repeats CreatedAt carries no
	// information. See the bookmarks_same_timestamp fixture.
	if other.CreatedAt.Before(e.CreatedAt) {
		e.UpdatedAt = e.UpdatedAt.Add(UpdatedAt(e.CreatedAt))
		e.CreatedAt = other.CreatedAt
	} else if other.CreatedAt.After(e.CreatedAt) {
		e.UpdatedAt = e.UpdatedAt.Add(UpdatedAt(other.CreatedAt))
	}

	e.UpdatedAt = e.UpdatedAt.Merge(other.UpdatedAt)
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

	var lastVisitedAt *int64
	if unix, ok := e.LastVisitedAt.Get(); ok {
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
		UpdatedAt:     sortedUnix(e.UpdatedAt),
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

	e.CreatedAt = NewCreatedAt(s.CreatedAt)

	e.UpdatedAt = unixToSet(s.UpdatedAt)

	if s.LastVisitedAt != nil {
		e.LastVisitedAt = NewLastVisitedAt(*s.LastVisitedAt)
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
		CreatedAt: NewCreatedAt(createdAt.Unix()),
		UpdatedAt: make(Set[UpdatedAt]),
		Names:     names,
		Labels:    labels,
		Shared:    NewShared(p.Shared == "yes"),
		ToRead:    NewToRead(p.ToRead == "yes"),
		IsFeed:    NewIsFeed(false),
		Extended:  extended,
	}

	return entity, nil
}
