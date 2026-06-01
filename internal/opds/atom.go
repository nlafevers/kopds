package opds

import (
	"encoding/xml"
	"time"
)

const OPDSNamespace = "http://opds-spec.org/2010/catalog"

// Feed represents an Atom Feed element, the root of an OPDS catalog.
type Feed struct {
	XMLName xml.Name `xml:"http://www.w3.org/2005/Atom feed"`
	Opds    string   `xml:"xmlns:opds,attr"`

	ID      string    `xml:"id"`
	Title   string    `xml:"title"`
	Updated time.Time `xml:"updated"`

	Author  *Author  `xml:"author,omitempty"`
	Links   []Link   `xml:"link"`
	Entries []*Entry `xml:"entry"`
}

// Entry represents an individual catalog entry in an Atom Feed.
type Entry struct {
	ID      string    `xml:"id"`
	Title   string    `xml:"title"`
	Updated time.Time `xml:"updated"`

	Authors []Author `xml:"author,omitempty"`
	Content *Content `xml:"content,omitempty"`
	Summary *Content `xml:"summary,omitempty"`
	Links   []Link   `xml:"link"`
}

// Author represents an Atom author element.
type Author struct {
	Name string `xml:"name"`
	URI  string `xml:"uri,omitempty"`
}

// Link represents an Atom link element, used for navigation and acquisition.
type Link struct {
	Rel   string `xml:"rel,attr"`
	Type  string `xml:"type,attr,omitempty"`
	Href  string `xml:"href,attr"`
	Title string `xml:"title,attr,omitempty"`
}

// Content represents an Atom content or summary element.
type Content struct {
	Type string `xml:"type,attr,omitempty"`
	Text string `xml:",chardata"`
}

// NewFeed creates a new Feed with the standard OPDS 1.2 namespaces.
func NewFeed(title, id string, links []Link) Feed {
	return Feed{
		Opds:    OPDSNamespace,
		Title:   title,
		ID:      id,
		Updated: time.Now(),
		Links:   links,
	}
}
