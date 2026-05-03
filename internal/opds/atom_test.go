package opds

import (
	"testing"
)

func TestNewFeed(t *testing.T) {
	title := "Test Catalog"
	id := "test-id"
	links := []Link{
		{Rel: "self", Href: "/opds/v1.2/catalog.xml", Type: "application/atom+xml;profile=opds-catalog;kind=navigation"},
	}

	feed := NewFeed(title, id, links)

	if feed.Title != title {
		t.Errorf("expected title %s, got %s", title, feed.Title)
	}

	if feed.ID != id {
		t.Errorf("expected id %s, got %s", id, feed.ID)
	}

	if feed.Xmlns != AtomNamespace {
		t.Errorf("expected xmlns %s, got %s", AtomNamespace, feed.Xmlns)
	}

	if feed.Opds != OPDSNamespace {
		t.Errorf("expected opds xmlns %s, got %s", OPDSNamespace, feed.Opds)
	}

	if len(feed.Links) != 1 {
		t.Errorf("expected 1 link, got %d", len(feed.Links))
	}

	if feed.Updated.IsZero() {
		t.Error("expected Updated time to be set, got zero")
	}
}
