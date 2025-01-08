package timesketch

import "testing"

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
