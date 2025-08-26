package formatter

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/template"

	"github.com/henrytill/hbt-go/internal"
)

type HTMLFormatter struct{}

// NewHTMLFormatter creates a new HTML formatter
func NewHTMLFormatter() *HTMLFormatter {
	return &HTMLFormatter{}
}

func (f *HTMLFormatter) Format(writer io.Writer, collection *internal.Collection) error {
	// HTML template for Netscape bookmark format
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

	// Prepare template data
	templateData := struct {
		Entities []templateEntity
	}{
		Entities: make([]templateEntity, 0, len(collection.Value)),
	}

	for _, node := range collection.Value {
		entity := templateEntity{
			URI:           node.Entity.URI,
			Title:         getFirstName(node.Entity.Names),
			CreatedAt:     node.Entity.CreatedAt,
			Labels:        internal.MapToSortedSlice(node.Entity.Labels),
			Shared:        node.Entity.Shared,
			ToRead:        node.Entity.ToRead,
			IsFeed:        node.Entity.IsFeed,
			LastVisitedAt: node.Entity.LastVisitedAt,
			Extended:      node.Entity.Extended,
		}

		// Set LastModified if there are UpdatedAt values
		if len(node.Entity.UpdatedAt) > 0 {
			entity.LastModified = &node.Entity.UpdatedAt[0]
		}

		// Sort labels for consistent output
		sort.Strings(entity.Labels)
		entity.TagsString = strings.Join(entity.Labels, ",")

		templateData.Entities = append(templateData.Entities, entity)
	}

	// Parse and execute template
	t, err := template.New("html").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	return t.Execute(writer, templateData)
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

func getFirstName(names map[string]struct{}) string {
	if len(names) == 0 {
		return ""
	}
	// Get first name alphabetically (for consistent ordering)
	keys := make([]string, 0, len(names))
	for k := range names {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys[0]
}
