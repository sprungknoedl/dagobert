package timesketch

import (
	"testing"

	"github.com/sprungknoedl/dagobert/pkg/fp"
)

func TestNewClient(t *testing.T) {
	c, err := NewClient("https://timesketch.lolcathost.io", "tom", "fZabMXXw-abbNd6j")
	t.Logf("csrf token = %s", c.csrfToken)

	if err != nil {
		t.Fatal(err)
	}
}

func TestListSketches(t *testing.T) {
	c, err := NewClient("https://timesketch.lolcathost.io", "tom", "fZabMXXw-abbNd6j")
	if err != nil {
		t.Fatal(err)
	}

	sketches, err := c.ListSketches()
	if err != nil {
		t.Fatal(err)
	}

	if len(sketches) != 1 {
		t.Errorf("expected %d sketches, got %d", 1, len(sketches))
	}
}

func TestUpload(t *testing.T) {
	c, err := NewClient("https://timesketch.lolcathost.io", "tom", "fZabMXXw-abbNd6j")
	if err != nil {
		t.Fatal(err)
	}

	err = c.Upload(1, "/Users/tom/Downloads/dummy.jsonl")
	if err != nil {
		t.Fatal(err)
	}
}

func TestExplore(t *testing.T) {
	c, err := NewClient("https://timesketch.lolcathost.io", "tom", "fZabMXXw-abbNd6j")
	if err != nil {
		t.Fatal(err)
	}

	sketch, err := c.GetSketch(1)
	if err != nil {
		t.Fatal(err)
	}

	events, err := c.Explore(1, "*", Filter{
		Size:    1024,
		Order:   "asc",
		Indices: fp.Apply(sketch.Timelines, func(t Timeline) int { return t.ID }),
		Chips:   []Chip{StarredEventsChip},
		Fields:  sketch.Mappings,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(events) < 10 {
		t.Errorf("expected >%d events, got %d", 10, len(events))
	}
}
