package types

import (
	"slices"
	"testing"
	"time"

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
	early := NewLastVisitedAt(time.Unix(100, 0))
	late := NewLastVisitedAt(time.Unix(200, 0))

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

func entityAt(uri string, unix int64) Entity {
	return Entity{
		URI:       mustParseURL(uri),
		CreatedAt: CreatedAt(time.Unix(unix, 0)),
		UpdatedAt: []UpdatedAt{},
		Names:     make(map[Name]struct{}),
		Labels:    make(map[Label]struct{}),
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
	first.Extended = []Extended{"one"}

	second := entityAt("https://example.com/", 200)
	second.Names[Name("Second")] = struct{}{}
	second.Labels[Label("b")] = struct{}{}
	second.Shared = NewShared(true)
	second.ToRead = NewToRead(true)
	second.Extended = []Extended{"two"}
	second.LastVisitedAt = NewLastVisitedAt(time.Unix(300, 0))

	idFirst := coll.Upsert(first)
	idSecond := coll.Upsert(second)

	if coll.Len() != 1 {
		t.Fatalf("Len = %d, want 1 after merging", coll.Len())
	}
	if idFirst != idSecond {
		t.Error("upserting the same URI should return the same id")
	}

	got := coll.Entities()[0]

	if names := MapToSortedSlice(got.Names); !slices.Equal(names, []string{"First", "Second"}) {
		t.Errorf("Names = %v, want union [First Second]", names)
	}
	if labels := MapToSortedSlice(got.Labels); !slices.Equal(labels, []string{"a", "b"}) {
		t.Errorf("Labels = %v, want union [a b]", labels)
	}
	if s, ok := got.Shared.Get(); !ok || !s {
		t.Errorf("Shared = (%v, %v), want (true, true): valid values OR together", s, ok)
	}
	if tr, ok := got.ToRead.Get(); !ok || !tr {
		t.Errorf("ToRead = (%v, %v), want (true, true): set value wins over unset", tr, ok)
	}
	if len(got.Extended) != 2 {
		t.Errorf("Extended = %v, want both values concatenated", got.Extended)
	}
	if lv, ok := got.LastVisitedAt.Get(); !ok || !lv.Equal(time.Unix(300, 0)) {
		t.Errorf("LastVisitedAt = (%v, %v), want (300, true)", lv, ok)
	}
}

func TestUpsertKeepsEarliestCreatedAt(t *testing.T) {
	t.Run("later entity recorded as update", func(t *testing.T) {
		coll := NewCollection()
		coll.Upsert(entityAt("https://example.com/", 100))
		coll.Upsert(entityAt("https://example.com/", 200))

		got := coll.Entities()[0]
		if got.CreatedAt.Unix() != 100 {
			t.Errorf("CreatedAt = %d, want 100 (earliest)", got.CreatedAt.Unix())
		}
		if len(got.UpdatedAt) != 1 || got.UpdatedAt[0].Unix() != 200 {
			t.Errorf("UpdatedAt = %v, want [200]", got.UpdatedAt)
		}
	})

	t.Run("earlier entity becomes creation, old time recorded as update", func(t *testing.T) {
		coll := NewCollection()
		coll.Upsert(entityAt("https://example.com/", 200))
		coll.Upsert(entityAt("https://example.com/", 100))

		got := coll.Entities()[0]
		if got.CreatedAt.Unix() != 100 {
			t.Errorf("CreatedAt = %d, want 100 (earliest)", got.CreatedAt.Unix())
		}
		if len(got.UpdatedAt) != 1 || got.UpdatedAt[0].Unix() != 200 {
			t.Errorf("UpdatedAt = %v, want [200]", got.UpdatedAt)
		}
	})

	t.Run("updates stay sorted", func(t *testing.T) {
		coll := NewCollection()
		coll.Upsert(entityAt("https://example.com/", 300))
		coll.Upsert(entityAt("https://example.com/", 100))
		coll.Upsert(entityAt("https://example.com/", 200))

		got := coll.Entities()[0]
		if got.CreatedAt.Unix() != 100 {
			t.Errorf("CreatedAt = %d, want 100 (earliest)", got.CreatedAt.Unix())
		}
		updates := make([]int64, len(got.UpdatedAt))
		for i, u := range got.UpdatedAt {
			updates[i] = u.Unix()
		}
		if !slices.Equal(updates, []int64{200, 300}) {
			t.Errorf("UpdatedAt = %v, want [200 300] sorted ascending", updates)
		}
	})

	t.Run("identical timestamp is not recorded as update", func(t *testing.T) {
		coll := NewCollection()
		coll.Upsert(entityAt("https://example.com/", 100))
		coll.Upsert(entityAt("https://example.com/", 100))

		got := coll.Entities()[0]
		if len(got.UpdatedAt) != 0 {
			t.Errorf("UpdatedAt = %v, want empty for identical timestamps", got.UpdatedAt)
		}
	})
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

	labels := MapToSortedSlice(coll.Entities()[0].Labels)
	if !slices.Equal(labels, []string{"keep", "new"}) {
		t.Errorf("Labels = %v, want [keep new]", labels)
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
		if names := MapToSortedSlice(entity.Names); !slices.Equal(names, []string{"Example"}) {
			t.Errorf("Names = %v, want trimmed [Example]", names)
		}
		if labels := MapToSortedSlice(entity.Labels); !slices.Equal(labels, []string{"go", "web"}) {
			t.Errorf("Labels = %v, want [go web]", labels)
		}
		if s, ok := entity.Shared.Get(); !ok || !s {
			t.Errorf("Shared = (%v, %v), want (true, true)", s, ok)
		}
		if tr, ok := entity.ToRead.Get(); !ok || tr {
			t.Errorf("ToRead = (%v, %v), want (false, true)", tr, ok)
		}
		if len(entity.Extended) != 1 || entity.Extended[0] != "extended text" {
			t.Errorf("Extended = %v, want trimmed [extended text]", entity.Extended)
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
