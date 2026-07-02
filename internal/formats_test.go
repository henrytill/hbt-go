package internal

import "testing"

func TestDetectInputFormat(t *testing.T) {
	tests := []struct {
		filename string
		want     Format
		found    bool
	}{
		{"bookmarks.html", HTML, true},
		{"bookmarks.HTML", HTML, true},
		{"posts.json", JSON, true},
		{"posts.xml", XML, true},
		{"notes.md", Markdown, true},
		{"archive.tar.md", Markdown, true},
		{"noextension", Format{}, false},
		{"trailing.", Format{}, false},
		{"file.txt", Format{}, false},
		{"", Format{}, false},
	}

	for _, tt := range tests {
		got, found := DetectInputFormat(tt.filename)
		if got != tt.want || found != tt.found {
			t.Errorf("DetectInputFormat(%q) = (%v, %v), want (%v, %v)",
				tt.filename, got, found, tt.want, tt.found)
		}
	}
}

func TestDetectOutputFormat(t *testing.T) {
	tests := []struct {
		filename string
		want     Format
		found    bool
	}{
		{"out.html", HTML, true},
		{"out.yaml", YAML, true},
		{"out.YAML", YAML, true},
		{"noextension", Format{}, false},
		{"out.json", Format{}, false},
		{"", Format{}, false},
	}

	for _, tt := range tests {
		got, found := DetectOutputFormat(tt.filename)
		if got != tt.want || found != tt.found {
			t.Errorf("DetectOutputFormat(%q) = (%v, %v), want (%v, %v)",
				tt.filename, got, found, tt.want, tt.found)
		}
	}
}
