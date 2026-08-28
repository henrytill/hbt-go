package types

import (
	"net/url"
	"slices"
	"testing"
	"time"
)

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func makeEntity(uri string) Entity {
	return Entity{
		URI:       mustParseURL(uri),
		CreatedAt: CreatedAt(time.Time{}),
		UpdatedAt: []UpdatedAt{},
		Names:     make(Set[Name]),
		Labels:    make(Set[Label]),
	}
}

func TestAddEdges_wrongCollection_panics(t *testing.T) {
	collA := NewCollection()
	collB := NewCollection()

	idA := collA.Upsert(makeEntity("https://example.com/a"))
	idB := collB.Upsert(makeEntity("https://example.com/b"))

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on foreign id")
		}
	}()

	collA.AddEdges(idA, idB)
}

func TestAddEdges_repeatedRelationRecordsOneEdge(t *testing.T) {
	coll := NewCollection()

	foo := coll.Upsert(makeEntity("https://foo.com"))
	bar := coll.Upsert(makeEntity("https://bar.com"))
	baz := coll.Upsert(makeEntity("https://baz.com"))

	coll.AddEdges(foo, bar)
	coll.AddEdges(foo, baz)
	coll.AddEdges(foo, bar)
	coll.AddEdges(foo, bar)

	// Stating a relation three times must not record it three times, and
	// deduping must not collapse the distinct Foo-Baz edge along with it.
	want := map[Id][]uint{
		foo: {bar.index, baz.index},
		bar: {foo.index},
		baz: {foo.index},
	}
	for id, edges := range want {
		if got := coll.edges[id.index]; !slices.Equal(got, edges) {
			t.Errorf("node %d: edges = %v, want %v", id.index, got, edges)
		}
	}
}
