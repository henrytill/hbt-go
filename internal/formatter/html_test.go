package formatter

import (
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/henrytill/hbt-go/internal/parser"
	"github.com/henrytill/hbt-go/internal/types"
)

func specialEntity(t *testing.T) types.Entity {
	t.Helper()
	u, err := url.Parse("https://example.com/?a=1&b=2")
	if err != nil {
		t.Fatal(err)
	}
	return types.Entity{
		URI:       u,
		CreatedAt: types.CreatedAt(time.Unix(100, 0)),
		UpdatedAt: []types.UpdatedAt{},
		Names: map[types.Name]struct{}{
			types.Name(`Title with "quotes" & <markup>`): {},
		},
		Labels: map[types.Label]struct{}{
			types.Label("tag&co"): {},
		},
		Extended: []types.Extended{
			types.Extended(`description with <b>html</b> & "quotes"`),
		},
	}
}

func formatCollection(t *testing.T, coll *types.Collection) string {
	t.Helper()
	var buf strings.Builder
	f := &HTMLFormatter{}
	if err := f.Format(&buf, coll); err != nil {
		t.Fatalf("Format: %v", err)
	}
	return buf.String()
}

func TestHTMLFormatterEscapesSpecialCharacters(t *testing.T) {
	coll := types.NewCollection()
	coll.Upsert(specialEntity(t))

	out := formatCollection(t, &coll)

	wantEscaped := []string{
		`HREF="https://example.com/?a=1&amp;b=2"`,
		`TAGS="tag&amp;co"`,
		`>Title with "quotes" &amp; &lt;markup&gt;</A>`,
		`<DD>description with &lt;b&gt;html&lt;/b&gt; &amp; "quotes"`,
	}
	for _, want := range wantEscaped {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\noutput:\n%s", want, out)
		}
	}

	wantAbsent := []string{
		`&b=2"`,
		`<markup>`,
		`<b>html</b>`,
	}
	for _, unwanted := range wantAbsent {
		if strings.Contains(out, unwanted) {
			t.Errorf("output contains unescaped %q\noutput:\n%s", unwanted, out)
		}
	}
}

func TestHTMLFormatterPreservesSingleQuotes(t *testing.T) {
	u, err := url.Parse("https://example.com/")
	if err != nil {
		t.Fatal(err)
	}
	coll := types.NewCollection()
	coll.Upsert(types.Entity{
		URI:       u,
		CreatedAt: types.CreatedAt(time.Unix(100, 0)),
		Names:     map[types.Name]struct{}{types.Name("O'Reilly Radar"): {}},
	})

	out := formatCollection(t, &coll)

	if !strings.Contains(out, ">O'Reilly Radar</A>") {
		t.Errorf("single quotes should pass through unescaped\noutput:\n%s", out)
	}
}

func TestHTMLFormatterRoundTrip(t *testing.T) {
	coll := types.NewCollection()
	original := specialEntity(t)
	coll.Upsert(original)

	out := formatCollection(t, &coll)

	p := &parser.HTMLParser{}
	reparsed, err := p.Parse(strings.NewReader(out))
	if err != nil {
		t.Fatalf("reparsing formatted output: %v", err)
	}
	if reparsed.Len() != 1 {
		t.Fatalf("expected 1 entity after round trip, got %d", reparsed.Len())
	}

	got := slices.Collect(reparsed.Entities())[0]
	if got.URI.String() != original.URI.String() {
		t.Errorf("URI: got %q, want %q", got.URI.String(), original.URI.String())
	}
	for name := range original.Names {
		if _, ok := got.Names[name]; !ok {
			t.Errorf("name %q lost in round trip, got %v", name, got.Names)
		}
	}
	for label := range original.Labels {
		if _, ok := got.Labels[label]; !ok {
			t.Errorf("label %q lost in round trip, got %v", label, got.Labels)
		}
	}
	if len(got.Extended) != 1 || got.Extended[0] != original.Extended[0] {
		t.Errorf("extended: got %v, want %v", got.Extended, original.Extended)
	}
}
