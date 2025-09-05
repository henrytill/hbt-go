package formatter

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/template"

	"github.com/henrytill/hbt-go/internal/types"
)

type HTMLFormatter struct{}

func NewHTMLFormatter() *HTMLFormatter {
	return &HTMLFormatter{}
}

type templateEntity struct {
	URI           string
	Title         string
	CreatedAt     int64
	LastModified  *int64
	Labels        []string
	TagsString    string
	Shared        bool
	ToRead        bool
	IsFeed        bool
	LastVisitedAt *int64
	Extended      *string
}

func getFirstName(names map[Name]struct{}, def string) string {
	if len(names) == 0 {
		return def
	}
	keys := make([]string, 0, len(names))
	for k := range names {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)
	return keys[0]
}

func (f *HTMLFormatter) Format(writer io.Writer, collection *types.Collection) error {
	const tmpl = `<!DOCTYPE NETSCAPE-Bookmark-file-1>
<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">
<TITLE>Bookmarks</TITLE>
<H1>Bookmarks</H1>
<DL><p>
{{- range .Entities}}
    <DT><A HREF="{{.URI}}"{{if .CreatedAt}} ADD_DATE="{{.CreatedAt}}"{{end}}{{if .LastModified}} LAST_MODIFIED="{{.LastModified}}"{{end}}{{if .Labels}} TAGS="{{.TagsString}}"{{end}}{{if not .Shared}} PRIVATE="1"{{end}}{{if .LastVisitedAt}} LAST_VISIT="{{.LastVisitedAt}}"{{end}}{{if .ToRead}} TOREAD="1"{{end}}{{if .IsFeed}} FEED="true"{{end}}>{{.Title}}</A>
{{- if .Extended}}
    <DD>{{.Extended}}
{{- end}}
{{- end}}
</DL><p>
`

	templateData := struct {
		Entities []templateEntity
	}{
		Entities: make([]templateEntity, 0, len(collection.Value)),
	}

	for _, node := range collection.Value {
		var uriString string
		if node.Entity.URI != nil {
			uriString = node.Entity.URI.String()
		}
		var lastVisitedAtUnix *int64
		if node.Entity.LastVisitedAt != nil {
			unix := node.Entity.LastVisitedAt.Unix()
			lastVisitedAtUnix = &unix
		}

		entity := templateEntity{
			URI:           uriString,
			Title:         getFirstName(node.Entity.Names, uriString),
			CreatedAt:     node.Entity.CreatedAt.Unix(),
			Labels:        types.MapToSortedSlice(node.Entity.Labels),
			Shared:        node.Entity.Shared,
			ToRead:        node.Entity.ToRead,
			IsFeed:        node.Entity.IsFeed,
			LastVisitedAt: lastVisitedAtUnix,
			Extended:      node.Entity.Extended,
		}

		if len(node.Entity.UpdatedAt) > 0 {
			unix := node.Entity.UpdatedAt[0].Unix()
			entity.LastModified = &unix
		}

		sort.Strings(entity.Labels)
		entity.TagsString = strings.Join(entity.Labels, ",")

		templateData.Entities = append(templateData.Entities, entity)
	}

	t, err := template.New("html").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	return t.Execute(writer, templateData)
}
