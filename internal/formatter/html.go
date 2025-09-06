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

func newTemplateEntity(entity types.Entity) templateEntity {
	var uriString string
	if entity.URI != nil {
		uriString = entity.URI.String()
	}

	var lastVisitedAtUnix *int64
	if entity.LastVisitedAt != nil {
		unix := entity.LastVisitedAt.Unix()
		lastVisitedAtUnix = &unix
	}

	labels := types.MapToSortedSlice(entity.Labels)
	sort.Strings(labels)

	ret := templateEntity{
		URI:           uriString,
		Title:         getFirstName(entity.Names, uriString),
		CreatedAt:     entity.CreatedAt.Unix(),
		Labels:        labels,
		TagsString:    strings.Join(labels, ","),
		Shared:        entity.Shared,
		ToRead:        entity.ToRead,
		IsFeed:        entity.IsFeed,
		LastVisitedAt: lastVisitedAtUnix,
		Extended:      entity.Extended,
	}

	if len(entity.UpdatedAt) > 0 {
		unix := entity.UpdatedAt[0].Unix()
		ret.LastModified = &unix
	}

	return ret
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
		Entities: make([]templateEntity, 0, collection.Len()),
	}

	for _, entity := range collection.Entities() {
		templateEntity := newTemplateEntity(entity)
		templateData.Entities = append(templateData.Entities, templateEntity)
	}

	t, err := template.New("html").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	return t.Execute(writer, templateData)
}
