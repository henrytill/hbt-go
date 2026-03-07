package types

import (
	"net/url"
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
		Names:     make(map[Name]struct{}),
		Labels:    make(map[Label]struct{}),
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
