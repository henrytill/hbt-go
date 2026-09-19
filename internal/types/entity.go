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

type UpdatedAt struct{ timestamp }

func NewUpdatedAt(unix int64) UpdatedAt { return UpdatedAt{timestamp(unix)} }

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

// Normalize drops an update that merely repeats CreatedAt.
//
// A timestamp equal to CreatedAt carries no information that CreatedAt does
// not (#57). An update strictly below it is a different thing and is
// untouched: henrytill/hbt-data#34.
//
// This is the whole of the normal form (henrytill/hbt-data#38), and three
// places maintain it -- the three that take a history from input. absorb ends
// here, so a merge that demotes the later creation time to an update does not
// then record the earlier one twice. fromRepr ends here because a serialized
// history is input like any other. The HTML parser ends here because it reads
// ADD_DATE and LAST_MODIFIED independently, so one anchor may state the same
// instant in both -- html/bookmarks_simple. It is exported for that third
// caller, which lives in internal/parser.
//
// The Markdown parser and NewEntityFromPost are normal for a weaker reason:
// they record no updates at all. One that learns to must normalize too --
// Entity's fields are exported by decision (see AGENTS.md), so nothing but
// this note enforces it.
//
// Three call sites is a choice, not a discovered fact. Collection.insert is a
// narrower funnel: every production parser reaches it through Upsert, so
// normalizing there would cover the parse path in one place, let this method
// be unexported, and leave absorb's equality guard with nothing to protect
// that a Collection can hold.
// It was not taken because it changes what Upsert means -- from "store what
// you were given" to "store the normal form" -- and Collection.fromRepr
// assigns entities directly rather than through insert, so the decode half
// would still need its own call. Raised in review of the PR that added this;
// revisit it there rather than re-deriving it.
func (e *Entity) Normalize() {
	delete(e.UpdatedAt, UpdatedAt(e.CreatedAt))
}

// LatestUpdate returns the most recent recorded update instant as a Unix second
// count, and whether there is one.
func (e Entity) LatestUpdate() (int64, bool) {
	var latest int64
	found := false
	for u := range e.UpdatedAt {
		if unix := u.unix(); !found || unix > latest {
			latest, found = unix, true
		}
	}
	return latest, found
}

// mergedUpdates returns the creation time and the update history of a and b
// merged: both histories and both creation times.
//
// Putting the operands' creation times back into the history, and leaving it
// to Normalize to take the winner back out, is what makes absorbing
// associative. Every merge does it, so however a sequence of mentions is
// bracketed the result is every history and every creation time in it, minus
// the smallest -- which Normalize removes, so the rule has one spelling rather
// than two. Removing the winner only when the two creation times differ is not
// associative, and neither is removing every update at or below the winner;
// henrytill/hbt-data#36 has both counterexamples and pins this rule with
// bookmarks_merged_repeat, bookmarks_update_before_creation and
// bookmarks_incoming_update. An update equal to the winner merely repeats it
// (#57, bookmarks_same_timestamp); one strictly below it stays, a shape HTML
// states by reading ADD_DATE and LAST_MODIFIED independently.
//
// Like Set.Merge, this may reuse a's set rather than allocating.
func mergedUpdates(a, b Entity) (CreatedAt, Set[UpdatedAt]) {
	created := a.CreatedAt
	if b.CreatedAt.Before(created) {
		created = b.CreatedAt
	}

	updates := a.UpdatedAt.Merge(b.UpdatedAt).
		Add(UpdatedAt(a.CreatedAt)).
		Add(UpdatedAt(b.CreatedAt))

	return created, updates
}

// absorb merges other into e: field-wise, by the rule mergedUpdates states for
// the timestamps and by union or comparison for every other field, then
// Normalize. Do not give mergedUpdates a removal of its own -- two spellings of
// one rule is what a later change would have to keep in step.
func (e *Entity) absorb(other Entity) {
	// Absorbing an identical entity is a no-op, and that is not redundant: the
	// rule strips an update equal to CreatedAt, so for an entity whose history
	// repeats its own creation time the same mention twice would not read like
	// it once.
	//
	// Normalizing at the parse and decode boundaries means such an entity no
	// longer arrives from input -- html/bookmarks_simple used to parse to that
	// shape and no longer does -- but Entity's fields are exported, so any
	// package can still write one. TestUpsertIdenticalEntityIsNoOp does, and
	// removing this guard fails that test and nothing else.
	//
	// hbt-hs, hbt-rs and hbt-ocaml guard the same way, and in all four the
	// guard still changes results -- what differs is only how far the shape
	// reaches. Here and in hbt-hs the fields are exported, so any package or
	// module can write it. hbt-ocaml's type is abstract, but its make takes an
	// ?updated_at it deliberately does not normalize, so any caller can too.
	// hbt-rs is the closed one: its fields are private, so only its own module
	// can -- which its tests do, and its merge doc says the guard is what keeps
	// that update. None of the four is inert.
	if e.Equal(other) {
		return
	}

	e.CreatedAt, e.UpdatedAt = mergedUpdates(*e, other)

	e.Names = e.Names.Merge(other.Names)
	e.Labels = e.Labels.Merge(other.Labels)
	e.Extended = e.Extended.Merge(other.Extended)

	e.Shared = e.Shared.Merge(other.Shared)
	e.ToRead = e.ToRead.Merge(other.ToRead)
	e.IsFeed = e.IsFeed.Merge(other.IsFeed)

	e.LastVisitedAt = e.LastVisitedAt.Merge(other.LastVisitedAt)

	e.Normalize()
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

	// A serialized history is input like any other, so decoding must not
	// reintroduce an entity whose UpdatedAt holds its CreatedAt.
	//
	// No fixture can pin this half. YAML is output-only here, and the JSON
	// input format is Pinboard JSON rather than a serialized collection, so no
	// CLI path reaches fromRepr -- it is entered only through Collection's
	// UnmarshalYAML/UnmarshalJSON. TestEntityFromReprNormalizes covers it.
	// (hbt-ocaml differs: it does accept -f yaml, so there the same call is on
	// a real CLI path.)
	e.Normalize()

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
		Names:     names,
		Labels:    labels,
		Shared:    NewShared(p.Shared == "yes"),
		ToRead:    NewToRead(p.ToRead == "yes"),
		IsFeed:    NewIsFeed(false),
		Extended:  extended,
	}

	return entity, nil
}
