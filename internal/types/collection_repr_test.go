package types

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

func makeReprTestCollection(t *testing.T) Collection {
	t.Helper()

	coll := NewCollection()

	parent := Entity{
		URI:           mustParseURL("https://example.com/parent"),
		CreatedAt:     NewCreatedAt(100),
		UpdatedAt:     NewSet(NewUpdatedAt(200)),
		Names:         NewSet[Name]("Parent"),
		Labels:        NewSet[Label]("a", "b"),
		Shared:        NewShared(true),
		ToRead:        NewToRead(false),
		IsFeed:        NewIsFeed(false),
		Extended:      NewSet[Extended]("extended text"),
		LastVisitedAt: NewLastVisitedAt(300),
	}
	child := Entity{
		URI:       mustParseURL("https://example.com/child"),
		CreatedAt: NewCreatedAt(150),
		UpdatedAt: make(Set[UpdatedAt]),
		Names:     NewSet[Name]("Child"),
		Labels:    NewSet[Label](),
	}

	parentID := coll.Upsert(parent)
	childID := coll.Upsert(child)
	coll.AddEdges(childID, parentID)

	return coll
}

func TestCollectionJSONRoundTrip(t *testing.T) {
	coll := makeReprTestCollection(t)

	first, err := json.Marshal(&coll)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got Collection
	if err := json.Unmarshal(first, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Len() != coll.Len() {
		t.Fatalf("Len after round trip: got %d, want %d", got.Len(), coll.Len())
	}

	second, err := json.Marshal(&got)
	if err != nil {
		t.Fatalf("Marshal after round trip: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("round trip not stable:\nfirst:  %s\nsecond: %s", first, second)
	}
}

func TestCollectionYAMLRoundTrip(t *testing.T) {
	coll := makeReprTestCollection(t)

	first, err := yaml.Marshal(&coll)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got Collection
	if err := yaml.Unmarshal(first, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Len() != coll.Len() {
		t.Fatalf("Len after round trip: got %d, want %d", got.Len(), coll.Len())
	}

	second, err := yaml.Marshal(&got)
	if err != nil {
		t.Fatalf("Marshal after round trip: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("round trip not stable:\nfirst:  %s\nsecond: %s", first, second)
	}
}

// An absent creation time is omitted from the wire rather than written as the epoch,
// and decodes back to an absence. Without that, an undated entity serialized as
// createdAt: 0 and decoded as one created on 1970-01-01, so the same collection merged
// differently depending on whether it had passed through YAML -- the serialization-only
// divergence henrytill/hbt-data#37 closes. A creation time of 0 is a real instant and
// still serializes.
func TestCollectionCreatedAtOmittedOnlyWhenAbsent(t *testing.T) {
	collectionWith := func(created CreatedAt) Collection {
		coll := NewCollection()
		coll.Upsert(Entity{
			URI:       mustParseURL("https://e.test/"),
			CreatedAt: created,
			UpdatedAt: make(Set[UpdatedAt]),
			Names:     NewSet[Name](),
			Labels:    NewSet[Label](),
		})
		return coll
	}

	encode := func(t *testing.T, marshal func(any) ([]byte, error), created CreatedAt) string {
		t.Helper()
		coll := collectionWith(created)
		data, err := marshal(&coll)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		return string(data)
	}

	marshalers := []struct {
		name    string
		marshal func(any) ([]byte, error)
		epoch   string
	}{
		{"json", json.Marshal, `"createdAt":0`},
		{"yaml", yaml.Marshal, "createdAt: 0"},
	}

	for _, m := range marshalers {
		t.Run(m.name, func(t *testing.T) {
			if out := encode(t, m.marshal, CreatedAt{}); strings.Contains(out, "createdAt") {
				t.Errorf("an undated entity should carry no createdAt:\n%s", out)
			}
			if out := encode(t, m.marshal, NewCreatedAt(0)); !strings.Contains(out, m.epoch) {
				t.Errorf("output missing %q:\n%s", m.epoch, out)
			}
		})
	}

	var decoded Collection
	if err := json.Unmarshal([]byte(encode(t, json.Marshal, CreatedAt{})), &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for e := range decoded.Entities() {
		if unix, ok := e.CreatedAt.Get(); ok {
			t.Errorf("CreatedAt = (%d, %v), want absent after a round trip", unix, ok)
		}
	}
}

func TestCollectionUnmarshalMalformed(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr string
	}{
		{
			name: "missing uri",
			data: `{"version":"0.1.0","length":1,"value":[
				{"id":0,"entity":{"uri":"","createdAt":100,"updatedAt":[],"names":[],"labels":[]},"edges":[]}]}`,
			wantErr: "missing uri",
		},
		{
			name: "edge out of range",
			data: `{"version":"0.1.0","length":1,"value":[
				{"id":0,"entity":{"uri":"https://example.com/","createdAt":100,"updatedAt":[],"names":[],"labels":[]},"edges":[7]}]}`,
			wantErr: "out of range",
		},
		{
			name:    "length mismatch",
			data:    `{"version":"0.1.0","length":3,"value":[]}`,
			wantErr: "length mismatch",
		},
		{
			name:    "invalid version",
			data:    `{"version":"bogus","length":0,"value":[]}`,
			wantErr: "invalid version",
		},
		{
			name:    "incompatible version",
			data:    `{"version":"9.9.9","length":0,"value":[]}`,
			wantErr: "incompatible version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var coll Collection
			err := json.Unmarshal([]byte(tt.data), &coll)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err, tt.wantErr)
			}
		})
	}
}
