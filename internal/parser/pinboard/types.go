package pinboard

import "github.com/henrytill/hbt-go/internal/types"

type Name = types.Name
type Label = types.Label

type Post struct {
	Href        string `xml:"href,attr"        json:"href"`
	Time        string `xml:"time,attr"        json:"time"`
	Description string `xml:"description,attr" json:"description"`
	Extended    string `xml:"extended,attr"    json:"extended"`
	Tags        string `xml:"tag,attr"         json:"tags"`
	Hash        string `xml:"hash,attr"        json:"hash"`
	Shared      string `xml:"shared,attr"      json:"shared"`
	ToRead      string `xml:"toread,attr"      json:"toread"`
}

type Posts struct {
	User  string `xml:"user,attr"`
	Posts []Post `xml:"post"`
}
