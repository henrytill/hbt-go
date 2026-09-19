package types

import (
	"slices"
	"testing"

	"github.com/henrytill/hbt-go/internal/pinboard"
)

func TestSharedMerge(t *testing.T) {
	unset := Shared{}
	no := NewShared(false)
	yes := NewShared(true)

	tests := []struct {
		name string
		a, b Shared
		want Shared
	}{
		{"unset absorbs other", unset, yes, yes},
		{"unset absorbs other false", unset, no, no},
		{"set keeps value over unset", yes, unset, yes},
		{"false keeps false over unset", no, unset, no},
		{"false or false", no, no, no},
		{"false or true", no, yes, yes},
		{"true or false", yes, no, yes},
		{"true or true", yes, yes, yes},
		{"both unset", unset, unset, unset},
	}

	for _, tt := range tests {
		if got := tt.a.Merge(tt.b); got != tt.want {
			t.Errorf("%s: %v.Merge(%v) = %v, want %v", tt.name, tt.a, tt.b, got, tt.want)
		}
	}
}

func TestToReadMerge(t *testing.T) {
	if got := NewToRead(false).Merge(NewToRead(true)); got != NewToRead(true) {
		t.Errorf("false.Merge(true) = %v, want true", got)
	}
	if got := (ToRead{}).Merge(NewToRead(false)); got != NewToRead(false) {
		t.Errorf("unset.Merge(false) = %v, want false", got)
	}
}

func TestIsFeedMerge(t *testing.T) {
	if got := NewIsFeed(true).Merge(NewIsFeed(false)); got != NewIsFeed(true) {
		t.Errorf("true.Merge(false) = %v, want true", got)
	}
	if got := (IsFeed{}).Merge(IsFeed{}); got != (IsFeed{}) {
		t.Errorf("unset.Merge(unset) = %v, want unset", got)
	}
}

func TestLastVisitedAtMerge(t *testing.T) {
	early := NewLastVisitedAt(100)
	late := NewLastVisitedAt(200)

	if got := early.Merge(late); got != late {
		t.Errorf("early.Merge(late) = %v, want late", got)
	}
	if got := late.Merge(early); got != late {
		t.Errorf("late.Merge(early) = %v, want late", got)
	}
	if got := (LastVisitedAt{}).Merge(early); got != early {
		t.Errorf("unset.Merge(early) = %v, want early", got)
	}
	if got := early.Merge(LastVisitedAt{}); got != early {
		t.Errorf("early.Merge(unset) = %v, want early", got)
	}
}

// An absent creation time is the identity, not a very old instant: an undated mention
// says nothing about when a bookmark was created, so it neither claims the creation time
// nor demotes a real one to an update (henrytill/hbt-data#37).
func TestCreatedAtMerge(t *testing.T) {
	absent := CreatedAt{}
	epoch := NewCreatedAt(0)
	early := NewCreatedAt(100)
	late := NewCreatedAt(200)

	tests := []struct {
		name string
		a, b CreatedAt
		want CreatedAt
	}{
		{"earlier wins", late, early, early},
		{"earlier wins whichever side states it", early, late, early},
		{"absent absorbs other", absent, early, early},
		{"set keeps value over absent", early, absent, early},
		{"both absent", absent, absent, absent},
		// The epoch is a real instant, so it survives an absence rather than
		// being read as one -- the distinction the zero value used to lose.
		{"absent absorbs the epoch", absent, epoch, epoch},
	}

	for _, tt := range tests {
		if got := tt.a.Merge(tt.b); !got.Equal(tt.want) {
			t.Errorf("%s: %v.Merge(%v) = %v, want %v", tt.name, tt.a, tt.b, got, tt.want)
		}
	}
}

func entityAt(uri string, unix int64) Entity {
	return Entity{
		URI:       mustParseURL(uri),
		CreatedAt: NewCreatedAt(unix),
		UpdatedAt: make(Set[UpdatedAt]),
		Names:     make(Set[Name]),
		Labels:    make(Set[Label]),
	}
}

// undatedEntityAt is what an anchor without ADD_DATE parses to: an entity with no
// creation time at all, which the zero value denotes.
func undatedEntityAt(uri string) Entity {
	e := entityAt(uri, 0)
	e.CreatedAt = CreatedAt{}
	return e
}

// createdUnix reports an entity's creation time as a Unix second count, or -1 if it has
// none -- a value no test below states, so reading an absence as an instant fails rather
// than passing as the epoch.
func createdUnix(e Entity) int64 {
	if unix, ok := e.CreatedAt.Get(); ok {
		return unix
	}
	return -1
}

func firstEntity(t *testing.T, coll Collection) Entity {
	t.Helper()
	for e := range coll.Entities() {
		return e
	}
	t.Fatal("collection is empty")
	return Entity{}
}

func TestLatestUpdate(t *testing.T) {
	e := entityAt("https://e.test/", 100)

	if unix, ok := e.LatestUpdate(); ok || unix != 0 {
		t.Errorf("LatestUpdate() = (%d, %v), want (0, false) for an entity with no updates", unix, ok)
	}

	e.UpdatedAt = NewSet(NewUpdatedAt(300), NewUpdatedAt(500), NewUpdatedAt(400))

	if unix, ok := e.LatestUpdate(); !ok || unix != 500 {
		t.Errorf("LatestUpdate() = (%d, %v), want (500, true)", unix, ok)
	}
}

func TestUpsertInsertsDistinctURIs(t *testing.T) {
	coll := NewCollection()

	idA := coll.Upsert(entityAt("https://example.com/a", 100))
	idB := coll.Upsert(entityAt("https://example.com/b", 200))

	if coll.Len() != 2 {
		t.Fatalf("Len = %d, want 2", coll.Len())
	}
	if idA == idB {
		t.Error("distinct URIs should get distinct ids")
	}
}

func TestUpsertMergesSameURI(t *testing.T) {
	coll := NewCollection()

	first := entityAt("https://example.com/", 100)
	first.Names[Name("First")] = struct{}{}
	first.Labels[Label("a")] = struct{}{}
	first.Shared = NewShared(false)
	first.Extended = NewSet[Extended]("one")

	second := entityAt("https://example.com/", 200)
	second.Names[Name("Second")] = struct{}{}
	second.Labels[Label("b")] = struct{}{}
	second.Shared = NewShared(true)
	second.ToRead = NewToRead(true)
	second.Extended = NewSet[Extended]("two")
	second.LastVisitedAt = NewLastVisitedAt(300)

	idFirst := coll.Upsert(first)
	idSecond := coll.Upsert(second)

	if coll.Len() != 1 {
		t.Fatalf("Len = %d, want 1 after merging", coll.Len())
	}
	if idFirst != idSecond {
		t.Error("upserting the same URI should return the same id")
	}

	got := firstEntity(t, coll)

	if names := SortedSlice(got.Names); !slices.Equal(names, []string{"First", "Second"}) {
		t.Errorf("Names = %v, want union [First Second]", names)
	}
	if labels := SortedSlice(got.Labels); !slices.Equal(labels, []string{"a", "b"}) {
		t.Errorf("Labels = %v, want union [a b]", labels)
	}
	if s, ok := got.Shared.Get(); !ok || !s {
		t.Errorf("Shared = (%v, %v), want (true, true): valid values OR together", s, ok)
	}
	if tr, ok := got.ToRead.Get(); !ok || !tr {
		t.Errorf("ToRead = (%v, %v), want (true, true): set value wins over unset", tr, ok)
	}
	if extended := SortedSlice(got.Extended); !slices.Equal(extended, []string{"one", "two"}) {
		t.Errorf("Extended = %v, want union [one two]", extended)
	}
	if lv, ok := got.LastVisitedAt.Get(); !ok || lv != 300 {
		t.Errorf("LastVisitedAt = (%v, %v), want (300, true)", lv, ok)
	}
}

func TestUpsertKeepsEarliestCreatedAt(t *testing.T) {
	t.Run("later entity recorded as update", func(t *testing.T) {
		coll := NewCollection()
		coll.Upsert(entityAt("https://example.com/", 100))
		coll.Upsert(entityAt("https://example.com/", 200))

		got := firstEntity(t, coll)
		if createdUnix(got) != 100 {
			t.Errorf("CreatedAt = %d, want 100 (earliest)", createdUnix(got))
		}
		if updates := sortedUnix(got.UpdatedAt); !slices.Equal(updates, []int64{200}) {
			t.Errorf("UpdatedAt = %v, want [200]", updates)
		}
	})

	t.Run("earlier entity becomes creation, old time recorded as update", func(t *testing.T) {
		coll := NewCollection()
		coll.Upsert(entityAt("https://example.com/", 200))
		coll.Upsert(entityAt("https://example.com/", 100))

		got := firstEntity(t, coll)
		if createdUnix(got) != 100 {
			t.Errorf("CreatedAt = %d, want 100 (earliest)", createdUnix(got))
		}
		if updates := sortedUnix(got.UpdatedAt); !slices.Equal(updates, []int64{200}) {
			t.Errorf("UpdatedAt = %v, want [200]", updates)
		}
	})

	// A merge that lowers CreatedAt onto an instant an earlier mention stated as its
	// LAST_MODIFIED used to leave that instant behind, repeating CreatedAt. Fixture:
	// bookmarks_superseded_creation. See #75.
	t.Run("update superseded by the lowered creation is dropped", func(t *testing.T) {
		coll := NewCollection()

		inverted := entityAt("https://example.com/", 200)
		inverted.UpdatedAt = NewSet(NewUpdatedAt(100))
		coll.Upsert(inverted)

		coll.Upsert(entityAt("https://example.com/", 100))

		got := firstEntity(t, coll)
		if createdUnix(got) != 100 {
			t.Errorf("CreatedAt = %d, want 100 (earliest)", createdUnix(got))
		}
		if updates := sortedUnix(got.UpdatedAt); !slices.Equal(updates, []int64{200}) {
			t.Errorf("UpdatedAt = %v, want [200]: the lowered CreatedAt is not also an update", updates)
		}
	})

	// The incoming entity's own history is kept, not discarded: a second mention can
	// state a LAST_MODIFIED of its own. Fixture: bookmarks_incoming_update.
	t.Run("the incoming history is kept", func(t *testing.T) {
		coll := NewCollection()
		coll.Upsert(entityAt("https://example.com/", 100))

		later := entityAt("https://example.com/", 200)
		later.UpdatedAt = NewSet(NewUpdatedAt(300))
		coll.Upsert(later)

		got := firstEntity(t, coll)
		if updates := sortedUnix(got.UpdatedAt); !slices.Equal(updates, []int64{200, 300}) {
			t.Errorf("UpdatedAt = %v, want [200 300]", updates)
		}
	})

	// An update that repeats a CreatedAt no merge moved goes too, which is what makes
	// absorbing associative. Fixture: bookmarks_merged_repeat. See henrytill/hbt-data#36.
	t.Run("an update repeating an unmoved creation is dropped", func(t *testing.T) {
		coll := NewCollection()

		repeat := entityAt("https://example.com/", 100)
		repeat.UpdatedAt = NewSet(NewUpdatedAt(100))
		coll.Upsert(repeat)
		coll.Upsert(entityAt("https://example.com/", 200))

		got := firstEntity(t, coll)
		if createdUnix(got) != 100 {
			t.Errorf("CreatedAt = %d, want 100", createdUnix(got))
		}
		if updates := sortedUnix(got.UpdatedAt); !slices.Equal(updates, []int64{200}) {
			t.Errorf("UpdatedAt = %v, want [200]: the repeat carries no information", updates)
		}
	})

	// Absorbing is associative, which is what decides the rule: henrytill/hbt-data#36.
	// The discriminating shape is a history holding an instant equal to its own
	// CreatedAt, which one anchor states by repeating ADD_DATE in LAST_MODIFIED.
	// hbt-hs and hbt-rs pin the same triple.
	//
	// What this catches is a return to "remove the winner only when the two
	// creation times differ", the narrow rule hbt-data#36 rejects: that gives
	// {200} bracketed left and {100, 200} bracketed right. It does not catch the
	// rule this repository had before that PR, which is associative on this
	// triple and wrong for another reason -- "an update repeating an unmoved
	// creation is dropped" above is the subtest that fails against it.
	t.Run("absorbing is associative", func(t *testing.T) {
		// Each mention is rebuilt per use: Upsert stores the entity as given and
		// Set.Merge mutates in place, so sharing one value between the two
		// bracketings would let the first mutate the second's operands.
		mention := func(unix int64, updates ...int64) Entity {
			e := entityAt("https://example.com/", unix)
			e.UpdatedAt = unixToSet(updates)
			return e
		}

		left := NewCollection()
		left.Upsert(mention(100, 100))
		left.Upsert(mention(100))
		left.Upsert(mention(200))

		inner := NewCollection()
		inner.Upsert(mention(100))
		inner.Upsert(mention(200))

		right := NewCollection()
		right.Upsert(mention(100, 100))
		right.Upsert(firstEntity(t, inner))

		if got, want := firstEntity(t, right), firstEntity(t, left); !got.Equal(want) {
			t.Errorf("a+(b+c) = %+v, (a+b)+c = %+v", got, want)
		}
	})

	t.Run("updates stay sorted", func(t *testing.T) {
		coll := NewCollection()
		coll.Upsert(entityAt("https://example.com/", 300))
		coll.Upsert(entityAt("https://example.com/", 100))
		coll.Upsert(entityAt("https://example.com/", 200))

		got := firstEntity(t, coll)
		if createdUnix(got) != 100 {
			t.Errorf("CreatedAt = %d, want 100 (earliest)", createdUnix(got))
		}
		updates := sortedUnix(got.UpdatedAt)
		if !slices.Equal(updates, []int64{200, 300}) {
			t.Errorf("UpdatedAt = %v, want [200 300] sorted ascending", updates)
		}
	})

	t.Run("identical timestamp is not recorded as update", func(t *testing.T) {
		coll := NewCollection()
		coll.Upsert(entityAt("https://example.com/", 100))
		coll.Upsert(entityAt("https://example.com/", 100))

		got := firstEntity(t, coll)
		if len(got.UpdatedAt) != 0 {
			t.Errorf("UpdatedAt = %v, want empty for identical timestamps", got.UpdatedAt)
		}
	})
}

// An undated mention says nothing about when the bookmark was created, so a dated one
// wins outright rather than being demoted to an update by an absence standing in as the
// epoch (henrytill/hbt-data#37). HTML states this shape by making ADD_DATE optional;
// fixtures html/bookmarks_undated, html/bookmarks_undated_merged and
// html/bookmarks_undated_both pin all three cases.
func TestUpsertUndatedMention(t *testing.T) {
	// Rebuilt per use: Upsert stores the entity as given and Set.Merge mutates in
	// place, so sharing one value between two bracketings would let the first
	// mutate the second's operands.
	undated := func(label Label) Entity {
		e := undatedEntityAt("https://example.com/")
		e.Labels[label] = struct{}{}
		return e
	}
	dated := func(unix int64, label Label) Entity {
		e := entityAt("https://example.com/", unix)
		e.Labels[label] = struct{}{}
		return e
	}

	t.Run("a dated mention wins over an undated one", func(t *testing.T) {
		coll := NewCollection()
		coll.Upsert(undated("undated"))
		coll.Upsert(dated(1609459200, "dated"))

		got := firstEntity(t, coll)
		if createdUnix(got) != 1609459200 {
			t.Errorf("CreatedAt = %d, want 1609459200 (the only stated instant)", createdUnix(got))
		}
		if len(got.UpdatedAt) != 0 {
			t.Errorf("UpdatedAt = %v, want empty: an absence is not an update", sortedUnix(got.UpdatedAt))
		}
	})

	// The other order agrees, which is what makes absence an identity rather than a value.
	t.Run("in either order", func(t *testing.T) {
		coll := NewCollection()
		coll.Upsert(dated(1609459200, "dated"))
		coll.Upsert(undated("undated"))

		got := firstEntity(t, coll)
		if createdUnix(got) != 1609459200 {
			t.Errorf("CreatedAt = %d, want 1609459200 (the only stated instant)", createdUnix(got))
		}
		if len(got.UpdatedAt) != 0 {
			t.Errorf("UpdatedAt = %v, want empty: an absence is not an update", sortedUnix(got.UpdatedAt))
		}
	})

	t.Run("two undated mentions stay undated", func(t *testing.T) {
		coll := NewCollection()
		coll.Upsert(undated("first"))
		coll.Upsert(undated("second"))

		got := firstEntity(t, coll)
		if _, ok := got.CreatedAt.Get(); ok {
			t.Errorf("CreatedAt = %d, want absent: merging two absences cannot invent an instant", createdUnix(got))
		}
		if len(got.UpdatedAt) != 0 {
			t.Errorf("UpdatedAt = %v, want empty", sortedUnix(got.UpdatedAt))
		}
		if labels := SortedSlice(got.Labels); !slices.Equal(labels, []string{"first", "second"}) {
			t.Errorf("Labels = %v, want union [first second]: the mentions still merged", labels)
		}
	})

	// Associativity has to survive an absence too, since Merge is the one place the
	// rule stops being a minimum: henrytill/hbt-data#36 decides the rule, #37 adds
	// the operand it has to hold for.
	t.Run("absorbing stays associative", func(t *testing.T) {
		left := NewCollection()
		left.Upsert(undated("a"))
		left.Upsert(dated(200, "b"))
		left.Upsert(dated(100, "c"))

		inner := NewCollection()
		inner.Upsert(dated(200, "b"))
		inner.Upsert(dated(100, "c"))

		right := NewCollection()
		right.Upsert(undated("a"))
		right.Upsert(firstEntity(t, inner))

		want := firstEntity(t, left)
		if got := firstEntity(t, right); !got.Equal(want) {
			t.Errorf("a+(b+c) = %+v, (a+b)+c = %+v", got, want)
		}
		if createdUnix(want) != 100 {
			t.Errorf("CreatedAt = %d, want 100 (earliest stated)", createdUnix(want))
		}
		if updates := sortedUnix(want.UpdatedAt); !slices.Equal(updates, []int64{200}) {
			t.Errorf("UpdatedAt = %v, want [200]", updates)
		}
	})
}

// An absent creation time is not the epoch. The zero value denoted both before
// henrytill/hbt-data#37, so an undated entity and one created on 1970-01-01 compared
// equal, merged alike, and serialized the same.
func TestEntityAbsentCreatedAtIsNotTheEpoch(t *testing.T) {
	undated := undatedEntityAt("https://e.test/")
	epoch := entityAt("https://e.test/", 0)

	if undated.Equal(epoch) || epoch.Equal(undated) {
		t.Error("an undated entity should not equal one created at the epoch")
	}
	if unix, ok := epoch.CreatedAt.Get(); !ok || unix != 0 {
		t.Errorf("CreatedAt = (%d, %v), want (0, true): 0 is a real instant", unix, ok)
	}
}

// The update equal to CreatedAt is what keeps this test load-bearing: that is the one
// element Normalize removes, so it is the only thing the identical-entity guard still
// changes. Since the parse path normalizes, an anchor stating LAST_MODIFIED == ADD_DATE
// no longer produces this shape -- html/bookmarks_simple used to and no longer does
// (henrytill/hbt-data#38). What still can is writing the field directly, as below;
// Entity's fields are exported by decision, so that stays reachable.
func TestUpsertIdenticalEntityIsNoOp(t *testing.T) {
	identical := func() Entity {
		e := entityAt("https://e.test/", 100)
		e.UpdatedAt = NewSet(NewUpdatedAt(100))
		e.Names[Name("a")] = struct{}{}
		e.Extended = NewSet[Extended]("desc")
		return e
	}

	coll := NewCollection()
	coll.Upsert(identical())
	coll.Upsert(identical())
	coll.Upsert(identical())

	got := firstEntity(t, coll)
	if extended := SortedSlice(got.Extended); !slices.Equal(extended, []string{"desc"}) {
		t.Errorf("Extended = %v, want [desc]: a repeated bookmark should not repeat its description", extended)
	}
	if updates := sortedUnix(got.UpdatedAt); !slices.Equal(updates, []int64{100}) {
		t.Errorf("UpdatedAt = %v, want [100]: the guard keeps what one mention stated", updates)
	}
	if names := SortedSlice(got.Names); !slices.Equal(names, []string{"a"}) {
		t.Errorf("Names = %v, want [a]", names)
	}
}

// Decoding normalizes, so a serialized collection cannot reintroduce an entity whose
// history repeats its own creation time. No fixture can pin this: YAML is output-only
// and the JSON input format is Pinboard JSON, not a serialized collection, so no CLI
// path reaches fromRepr.
//
// Both halves are asserted: an implementation dropping every update at or below
// CreatedAt would pass on the 100 alone, and only the 50 separates that from the rule
// that removes exactly CreatedAt (henrytill/hbt-data#34).
func TestEntityFromReprNormalizes(t *testing.T) {
	var e Entity
	createdAt := int64(100)
	if err := e.fromRepr(entityRepr{
		URI:       "https://e.test/",
		CreatedAt: &createdAt,
		UpdatedAt: []int64{50, 100, 300},
	}); err != nil {
		t.Fatalf("fromRepr: %v", err)
	}

	if got := createdUnix(e); got != 100 {
		t.Errorf("CreatedAt = %d, want 100", got)
	}
	if updates := sortedUnix(e.UpdatedAt); !slices.Equal(updates, []int64{50, 300}) {
		t.Errorf("UpdatedAt = %v, want [50 300]: the update equal to CreatedAt goes, the one below it stays", updates)
	}
}

func TestUpsertSharedDescriptionIsNotDuplicated(t *testing.T) {
	describedWithLabel := func(label Label) Entity {
		e := entityAt("https://e.test/", 100)
		e.Labels[label] = struct{}{}
		e.Extended = NewSet[Extended]("desc")
		return e
	}

	coll := NewCollection()
	coll.Upsert(describedWithLabel("a"))
	coll.Upsert(describedWithLabel("b"))
	coll.Upsert(describedWithLabel("c"))

	got := firstEntity(t, coll)
	if extended := SortedSlice(got.Extended); !slices.Equal(extended, []string{"desc"}) {
		t.Errorf("Extended = %v, want [desc]: entities that differ elsewhere still share one description", extended)
	}
	if labels := SortedSlice(got.Labels); !slices.Equal(labels, []string{"a", "b", "c"}) {
		t.Errorf("Labels = %v, want union [a b c]", labels)
	}
}

func TestUpsertSharedTimestampIsNotDuplicated(t *testing.T) {
	updatedWithLabel := func(unix int64, label Label) Entity {
		e := entityAt("https://e.test/", unix)
		e.Labels[label] = struct{}{}
		return e
	}

	coll := NewCollection()
	coll.Upsert(updatedWithLabel(300, "a"))
	coll.Upsert(updatedWithLabel(100, "b"))
	coll.Upsert(updatedWithLabel(300, "c"))
	coll.Upsert(updatedWithLabel(200, "d"))
	coll.Upsert(updatedWithLabel(300, "e"))

	got := firstEntity(t, coll)
	if createdUnix(got) != 100 {
		t.Errorf("CreatedAt = %d, want 100 (earliest)", createdUnix(got))
	}
	if updates := sortedUnix(got.UpdatedAt); !slices.Equal(updates, []int64{200, 300}) {
		t.Errorf("UpdatedAt = %v, want [200 300]: entities that differ elsewhere still share one timestamp", updates)
	}
}

func TestEntityEqual(t *testing.T) {
	base := func() Entity {
		e := entityAt("https://example.com/", 100)
		e.Names[Name("a")] = struct{}{}
		e.Labels[Label("l")] = struct{}{}
		e.UpdatedAt = NewSet(UpdatedAt{200})
		e.Shared = NewShared(true)
		e.Extended = NewSet[Extended]("desc")
		e.LastVisitedAt = NewLastVisitedAt(300)
		return e
	}

	if !base().Equal(base()) {
		t.Error("independently built identical entities should compare equal")
	}

	tests := []struct {
		name   string
		mutate func(*Entity)
	}{
		{"uri", func(e *Entity) { e.URI = mustParseURL("https://other.test/") }},
		{"nil uri", func(e *Entity) { e.URI = nil }},
		{"createdAt", func(e *Entity) { e.CreatedAt = NewCreatedAt(101) }},
		{"absent createdAt", func(e *Entity) { e.CreatedAt = CreatedAt{} }},
		{"updatedAt", func(e *Entity) { e.UpdatedAt[UpdatedAt{400}] = struct{}{} }},
		{"names", func(e *Entity) { e.Names[Name("b")] = struct{}{} }},
		{"labels", func(e *Entity) { delete(e.Labels, Label("l")) }},
		{"shared", func(e *Entity) { e.Shared = NewShared(false) }},
		{"toRead", func(e *Entity) { e.ToRead = NewToRead(false) }},
		{"isFeed", func(e *Entity) { e.IsFeed = NewIsFeed(false) }},
		{"extended", func(e *Entity) { e.Extended = NewSet[Extended]("other") }},
		{"lastVisitedAt", func(e *Entity) { e.LastVisitedAt = LastVisitedAt{} }},
	}

	for _, tt := range tests {
		modified := base()
		tt.mutate(&modified)
		if base().Equal(modified) {
			t.Errorf("%s: entities differing in %s should not compare equal", tt.name, tt.name)
		}
		if modified.Equal(base()) {
			t.Errorf("%s: equality should be symmetric", tt.name)
		}
	}
}

func TestApplyMappings(t *testing.T) {
	coll := NewCollection()
	e := entityAt("https://example.com/", 100)
	e.Labels[Label("old")] = struct{}{}
	e.Labels[Label("keep")] = struct{}{}
	e.Labels[Label("alias")] = struct{}{}
	coll.Upsert(e)

	coll.ApplyMappings(map[string]string{
		"old":   "new",
		"alias": "new", // two labels collapsing into one
	})

	labels := SortedSlice(firstEntity(t, coll).Labels)
	if !slices.Equal(labels, []string{"keep", "new"}) {
		t.Errorf("Labels = %v, want [keep new]", labels)
	}
}

func TestApplyMappingsDropsLabelsMappedToEmpty(t *testing.T) {
	coll := NewCollection()
	e := entityAt("https://example.com/", 100)
	e.Labels[Label("news")] = struct{}{}
	e.Labels[Label("go")] = struct{}{}
	coll.Upsert(e)

	coll.ApplyMappings(map[string]string{"news": ""})

	labels := SortedSlice(firstEntity(t, coll).Labels)
	if !slices.Equal(labels, []string{"go"}) {
		t.Errorf("Labels = %v, want [go]: a label mapped to the empty string is dropped, not replaced", labels)
	}
}

func TestNewEntityFromPost(t *testing.T) {
	t.Run("full post", func(t *testing.T) {
		entity, err := NewEntityFromPost(pinboard.Post{
			Href:        "https://example.com/",
			Time:        "2021-01-01T00:00:00Z",
			Description: "  Example  ",
			Extended:    " extended text ",
			Tags:        " go  web ",
			Shared:      "yes",
			ToRead:      "no",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if entity.URI.String() != "https://example.com/" {
			t.Errorf("URI = %s", entity.URI)
		}
		if names := SortedSlice(entity.Names); !slices.Equal(names, []string{"Example"}) {
			t.Errorf("Names = %v, want trimmed [Example]", names)
		}
		if labels := SortedSlice(entity.Labels); !slices.Equal(labels, []string{"go", "web"}) {
			t.Errorf("Labels = %v, want [go web]", labels)
		}
		if s, ok := entity.Shared.Get(); !ok || !s {
			t.Errorf("Shared = (%v, %v), want (true, true)", s, ok)
		}
		if tr, ok := entity.ToRead.Get(); !ok || tr {
			t.Errorf("ToRead = (%v, %v), want (false, true)", tr, ok)
		}
		if extended := SortedSlice(entity.Extended); !slices.Equal(extended, []string{"extended text"}) {
			t.Errorf("Extended = %v, want trimmed [extended text]", extended)
		}
	})

	t.Run("empty href is rejected", func(t *testing.T) {
		if _, err := NewEntityFromPost(pinboard.Post{Time: "2021-01-01T00:00:00Z"}); err == nil {
			t.Error("expected error for empty href")
		}
	})

	t.Run("malformed time is rejected", func(t *testing.T) {
		if _, err := NewEntityFromPost(pinboard.Post{Href: "https://example.com/", Time: "yesterday"}); err == nil {
			t.Error("expected error for malformed time")
		}
	})
}

func TestVersion(t *testing.T) {
	t.Run("accepts with and without v prefix", func(t *testing.T) {
		for _, s := range []string{"0.1.0", "v0.1.0"} {
			v, err := NewVersion(s)
			if err != nil {
				t.Fatalf("NewVersion(%q): %v", s, err)
			}
			if v.String() != "0.1.0" {
				t.Errorf("NewVersion(%q).String() = %q, want 0.1.0", s, v.String())
			}
		}
	})

	t.Run("rejects invalid semver", func(t *testing.T) {
		for _, s := range []string{"", "bogus", "1.2.3.4"} {
			if _, err := NewVersion(s); err == nil {
				t.Errorf("NewVersion(%q): expected error", s)
			}
		}
	})

	t.Run("compatibility is major.minor", func(t *testing.T) {
		tests := []struct {
			version string
			want    bool
		}{
			{"v0.1.0", true},
			{"v0.1.9", true},
			{"v0.2.0", false},
			{"v1.1.0", false},
		}
		for _, tt := range tests {
			v, err := NewVersion(tt.version)
			if err != nil {
				t.Fatalf("NewVersion(%q): %v", tt.version, err)
			}
			if got := v.IsCompatible(); got != tt.want {
				t.Errorf("IsCompatible(%q) = %v, want %v", tt.version, got, tt.want)
			}
		}
	})
}
