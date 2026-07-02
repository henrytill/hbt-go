package formatter

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/template"

	"github.com/henrytill/hbt-go/internal/types"
)

// attrEscaper escapes the characters that are unsafe inside the template's
// double-quoted attribute values. textEscaper does the same for text content,
// where quotes are safe and left alone. Neither escapes single quotes, both
// because they are safe in these contexts and to keep output byte-identical
// for existing bookmark files.
var (
	attrEscaper = strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	textEscaper = strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
)

type HTMLFormatter struct{}

// templateEntity holds the values interpolated into the bookmark template.
// The string fields sourced from entity data (Href, Text, Tags, Extended)
// are HTML-escaped by newTemplateEntity; the template must not escape again.
type templateEntity struct {
	Href         string
	Text         string
	AddDate      int64
	LastModified *int64
	Tags         string
	Private      string
	ToRead       string
	Feed         string
	LastVisit    *int64
	Extended     *string
}

func stringOfBool(b bool) string {
	if b {
		return "1"
	}
	return "0"
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
	if t, ok := entity.LastVisitedAt.Get(); ok {
		unix := t.Unix()
		lastVisit = &unix
	}

	tags := types.MapToSortedSlice(entity.Labels)

	var extended *string
	if len(entity.Extended) > 0 {
		s := textEscaper.Replace(string(entity.Extended[0]))
		extended = &s
	}

	var private string
	if shared, ok := entity.Shared.Get(); ok {
		private = stringOfBool(!shared)
	}

	var toRead string
	if tr, ok := entity.ToRead.Get(); ok {
		toRead = stringOfBool(tr)
	}

	var feed string
	if f, ok := entity.IsFeed.Get(); ok {
		feed = strconv.FormatBool(f)
	}

	ret := templateEntity{
		Href:      attrEscaper.Replace(href),
		Text:      textEscaper.Replace(text),
		AddDate:   entity.CreatedAt.Unix(),
		Tags:      attrEscaper.Replace(strings.Join(tags, ",")),
		Private:   private,
		ToRead:    toRead,
		Feed:      feed,
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
        {{- if .Private}} PRIVATE="{{.Private}}"{{end}}
        {{- if .LastVisit}} LAST_VISIT="{{.LastVisit}}"{{end}}
        {{- if .ToRead}} TOREAD="{{.ToRead}}"{{end}}
        {{- if .Feed}} FEED="{{.Feed}}"{{end}}>{{.Text}}</A>
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
