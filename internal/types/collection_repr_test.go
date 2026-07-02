package types

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
)

func makeReprTestCollection(t *testing.T) Collection {
	t.Helper()

	coll := NewCollection()

	parent := Entity{
		URI:           mustParseURL("https://example.com/parent"),
		CreatedAt:     CreatedAt(time.Unix(100, 0)),
		UpdatedAt:     []UpdatedAt{UpdatedAt(time.Unix(200, 0))},
		Names:         map[Name]struct{}{"Parent": {}},
		Labels:        map[Label]struct{}{"a": {}, "b": {}},
		Shared:        NewShared(true),
		ToRead:        NewToRead(false),
		IsFeed:        NewIsFeed(false),
		Extended:      []Extended{"extended text"},
		LastVisitedAt: NewLastVisitedAt(time.Unix(300, 0)),
	}
	child := Entity{
		URI:       mustParseURL("https://example.com/child"),
		CreatedAt: CreatedAt(time.Unix(150, 0)),
		UpdatedAt: []UpdatedAt{},
		Names:     map[Name]struct{}{"Child": {}},
		Labels:    map[Label]struct{}{},
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
