package formatter

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/henrytill/hbt-go/internal/types"
)

type HTMLFormatter struct{}

type templateEntity struct {
	Href         string
	Text         string
	AddDate      int64
	LastModified *int64
	Tags         string
	Private      bool
	ToRead       bool
	Feed         bool
	LastVisit    *int64
	Extended     *string
}

func newTemplateEntity(entity types.Entity) templateEntity {
	var href string
	if entity.URI != nil {
		href = entity.URI.String()
	}

	names := types.MapToSortedSlice(entity.Names)
	text := href
	if len(names) > 0 {
		text = names[0]
	}

	var lastVisit *int64
	if t, ok := entity.LastVisitedAt.Time(); ok {
		unix := t.Unix()
		lastVisit = &unix
	}

	tags := types.MapToSortedSlice(entity.Labels)

	var extended *string
	if len(entity.Extended) > 0 {
		s := string(entity.Extended[0])
		extended = &s
	}

	ret := templateEntity{
		Href:      href,
		Text:      text,
		AddDate:   entity.CreatedAt.Unix(),
		Tags:      strings.Join(tags, ","),
		Private:   !bool(entity.Shared),
		ToRead:    bool(entity.ToRead),
		Feed:      bool(entity.IsFeed),
		LastVisit: lastVisit,
		Extended:  extended,
	}

	if len(entity.UpdatedAt) > 0 {
		unix := entity.UpdatedAt[0].Unix()
		ret.LastModified = &unix
	}

	return ret
}

func (f *HTMLFormatter) Format(writer io.Writer, coll *types.Collection) error {
	const tmpl = `<!DOCTYPE NETSCAPE-Bookmark-file-1>
<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">
<TITLE>Bookmarks</TITLE>
<H1>Bookmarks</H1>
<DL><p>
{{- range .Entities}}
    <DT><A HREF="{{.Href}}"
        {{- if .AddDate}} ADD_DATE="{{.AddDate}}"{{end}}
        {{- if .LastModified}} LAST_MODIFIED="{{.LastModified}}"{{end}}
        {{- if .Tags}} TAGS="{{.Tags}}"{{end}}
        {{- if .Private}} PRIVATE="1"{{end}}
        {{- if .LastVisit}} LAST_VISIT="{{.LastVisit}}"{{end}}
        {{- if .ToRead}} TOREAD="1"{{end}}
        {{- if .Feed}} FEED="true"{{end}}>{{.Text}}</A>
{{- if .Extended}}
    <DD>{{.Extended}}
{{- end}}
{{- end}}
</DL><p>
`

	templateData := struct {
		Entities []templateEntity
	}{
		Entities: make([]templateEntity, 0, coll.Len()),
	}

	for _, entity := range coll.Entities() {
		templateEntity := newTemplateEntity(entity)
		templateData.Entities = append(templateData.Entities, templateEntity)
	}

	t, err := template.New("html").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	return t.Execute(writer, templateData)
}
