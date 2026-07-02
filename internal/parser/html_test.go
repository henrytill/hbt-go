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
	return coll.Entities()[0]
}

func labelSet(e types.Entity) map[string]struct{} {
	set := make(map[string]struct{})
	for label := range e.Labels {
		set[string(label)] = struct{}{}
	}
	return set
}

func TestHTMLParserToreadTag(t *testing.T) {
	t.Run("exact toread tag sets flag and is dropped from labels", func(t *testing.T) {
		e := parseSingleBookmark(t, `<A HREF="https://example.com/" ADD_DATE="100" TAGS="toread,go">Ex</A>`)

		if toRead, ok := e.ToRead.Get(); !ok || !toRead {
			t.Errorf("expected ToRead true, got (%v, %v)", toRead, ok)
		}
		labels := labelSet(e)
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
		labels := labelSet(e)
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
