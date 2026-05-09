package opds

import (
	"encoding/xml"
	"time"
)

const (
	AtomNamespace      = "http://www.w3.org/2005/Atom"
	OPDSNamespace      = "http://opds-spec.org/2010/catalog"
	ThreadingNamespace = "http://purl.org/syndication/thread/1.0"
)

// Feed represents an Atom Feed element, the root of an OPDS catalog.
type Feed struct {
	XMLName   xml.Name `xml:"http://www.w3.org/2005/Atom feed"`
	Opds      string   `xml:"xmlns:opds,attr"`
	Threading string   `xml:"xmlns:thr,attr"`

	ID      string    `xml:"id"`
	Title   string    `xml:"title"`
	Updated time.Time `xml:"updated"`
	Icon    string    `xml:"icon,omitempty"`

	Author  *Author  `xml:"author,omitempty"`
	Links   []Link   `xml:"link"`
	Entries []*Entry `xml:"entry"`
}

// Entry represents an individual catalog entry in an Atom Feed.
type Entry struct {
	ID        string    `xml:"id"`
	Title     string    `xml:"title"`
	Updated   time.Time `xml:"updated"`
	Published time.Time `xml:"published,omitempty"`

	Authors    []Author   `xml:"author,omitempty"`
	Content    *Content   `xml:"content,omitempty"`
	Summary    *Content   `xml:"summary,omitempty"`
	Links      []Link     `xml:"link"`
	Categories []Category `xml:"category,omitempty"`
	Rights     string     `xml:"rights,omitempty"`
}

// Author represents an Atom author element.
type Author struct {
	Name string `xml:"name"`
	URI  string `xml:"uri,omitempty"`
}

// Link represents an Atom link element, used for navigation and acquisition.
type Link struct {
	Rel        string `xml:"rel,attr"`
	Type       string `xml:"type,attr,omitempty"`
	Href       string `xml:"href,attr"`
	Title      string `xml:"title,attr,omitempty"`
	Count      int    `xml:"opds:count,attr,omitempty"`
	ThreadCount int    `xml:"http://purl.org/syndication/thread/1.0 count,attr,omitempty"`
	Price      string `xml:"opds:price,attr,omitempty"`
	Currency   string `xml:"opds:currency,attr,omitempty"`

	// Support for opds:indirectAcquisition to future-proof the acquisition pipeline.
	IndirectAcquisitions []IndirectAcquisition `xml:"http://opds-spec.org/2010/catalog indirectAcquisition,omitempty"`
}

// IndirectAcquisition represents an opds:indirectAcquisition element.
type IndirectAcquisition struct {
	Type                 string                `xml:"type,attr"`
	IndirectAcquisitions []IndirectAcquisition `xml:"indirectAcquisition,omitempty"`
}

// Category represents an Atom category element.
type Category struct {
	Term   string `xml:"term,attr"`
	Scheme string `xml:"scheme,attr,omitempty"`
	Label  string `xml:"label,attr,omitempty"`
}

// Content represents an Atom content or summary element.
type Content struct {
	Type string `xml:"type,attr,omitempty"`
	Text string `xml:",chardata"`
}

// NewFeed creates a new Feed with the standard OPDS 1.2 namespaces.
func NewFeed(title, id string, links []Link) Feed {
	return Feed{
		Opds:      OPDSNamespace,
		Threading: ThreadingNamespace,
		Title:     title,
		ID:        id,
		Updated:   time.Now(),
		Links:     links,
	}
}
