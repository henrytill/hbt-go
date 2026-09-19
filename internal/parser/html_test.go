package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/henrytill/hbt-go/internal/types"
)

func parseSingleBookmark(t *testing.T, anchor string) types.Entity {
	t.Helper()

	doc := fmt.Sprintf(`<!DOCTYPE NETSCAPE-Bookmark-file-1>
<DL><p>
    <DT>%s
</DL><p>
`, anchor)

	p := &HTMLParser{}
	coll, err := p.Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if coll.Len() != 1 {
		t.Fatalf("expected 1 entity, got %d", coll.Len())
	}
	for e := range coll.Entities() {
		return e
	}
	t.Fatal("no entity parsed")
	return types.Entity{}
}

// An anchor without ADD_DATE parses to an absent creation time: not the wall clock,
// which made output depend on when it was produced (#87), and not the epoch, which
// would win every merge and demote a real creation time to an update
// (henrytill/hbt-data#37). Fixture: html/bookmarks_undated.
func TestHTMLParserUndatedAnchor(t *testing.T) {
	undated := parseSingleBookmark(t, `<A HREF="https://example.com/" TAGS="a">Ex</A>`)
	if unix, ok := undated.CreatedAt.Get(); ok {
		t.Errorf("CreatedAt = (%d, %v), want absent", unix, ok)
	}

	// Only ADD_DATE's presence separates an absence from the epoch, which is a
	// real instant. Fixture: html/bookmarks_epoch_creation.
	epoch := parseSingleBookmark(t, `<A HREF="https://example.com/" ADD_DATE="0" TAGS="a">Ex</A>`)
	if unix, ok := epoch.CreatedAt.Get(); !ok || unix != 0 {
		t.Errorf("CreatedAt = (%d, %v), want (0, true)", unix, ok)
	}

	// An ADD_DATE that is not a timestamp states nothing usable, so it lands in the
	// same absence rather than in the wall clock it used to fall back to (#79).
	// Whether malformed input should instead be rejected is henrytill/hbt-data#11,
	// which the corpus cannot express yet.
	malformed := parseSingleBookmark(t, `<A HREF="https://example.com/" ADD_DATE="bogus" TAGS="a">Ex</A>`)
	if unix, ok := malformed.CreatedAt.Get(); ok {
		t.Errorf("CreatedAt = (%d, %v), want absent", unix, ok)
	}
}

func TestHTMLParserToreadTag(t *testing.T) {
	t.Run("exact toread tag sets flag and is dropped from labels", func(t *testing.T) {
		e := parseSingleBookmark(t, `<A HREF="https://example.com/" ADD_DATE="100" TAGS="toread,go">Ex</A>`)

		if toRead, ok := e.ToRead.Get(); !ok || !toRead {
			t.Errorf("expected ToRead true, got (%v, %v)", toRead, ok)
		}
		labels := e.Labels
		if _, exists := labels["toread"]; exists {
			t.Error("toread should not appear as a label")
		}
		if _, exists := labels["go"]; !exists {
			t.Error("expected go label to be kept")
		}
	})

	t.Run("tag merely containing toread is a plain label", func(t *testing.T) {
		e := parseSingleBookmark(t, `<A HREF="https://example.com/" ADD_DATE="100" TAGS="toreading,go">Ex</A>`)

		if _, ok := e.ToRead.Get(); ok {
			t.Error("ToRead should be unset for tag toreading")
		}
		labels := e.Labels
		if _, exists := labels["toreading"]; !exists {
			t.Error("expected toreading label to be kept")
		}
	})

	t.Run("TOREAD attribute takes precedence over tags", func(t *testing.T) {
		e := parseSingleBookmark(t, `<A HREF="https://example.com/" ADD_DATE="100" TAGS="toread" TOREAD="0">Ex</A>`)

		if toRead, ok := e.ToRead.Get(); !ok || toRead {
			t.Errorf("expected ToRead false from TOREAD attribute, got (%v, %v)", toRead, ok)
		}
	})

	t.Run("TOREAD attribute set without tags", func(t *testing.T) {
		e := parseSingleBookmark(t, `<A HREF="https://example.com/" ADD_DATE="100" TOREAD="1">Ex</A>`)

		if toRead, ok := e.ToRead.Get(); !ok || !toRead {
			t.Errorf("expected ToRead true from TOREAD attribute, got (%v, %v)", toRead, ok)
		}
	})
}
