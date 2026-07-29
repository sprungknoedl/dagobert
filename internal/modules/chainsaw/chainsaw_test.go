package chainsaw

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupports(t *testing.T) {
	m := &Module{}

	cases := []struct {
		name string
		obj  any
		want bool
	}{
		{"evtx passes", model.Evidence{Name: "Security.evtx"}, true},
		{"zip rejected", model.Evidence{Name: "triage.zip"}, false},
		{"no extension rejected", model.Evidence{Name: "README"}, false},
		{"non-evidence rejected", model.Indicator{Type: "Hash"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, m.Supports(tc.obj))
		})
	}
}

// rawIndividual and rawAggregate are real `chainsaw hunt --jsonl` output lines, recorded
// locally against chainsaw 2.16.2 with a sample scheduled-task .evtx: an "individual" hit
// carries a single "document", an "aggregate" hit (a Sigma `| count()` rule) carries a
// "documents" array instead.
const rawIndividual = `{"group":"Sigma","kind":"individual","document":{"kind":"evtx","path":"/evidence/sample.evtx","data":{"Event":{"System":{"EventID":4698}}}},"name":"Test Scheduled Task Created","timestamp":"2019-03-19T00:02:04.319945+00:00","authors":["unknown"],"level":"medium","source":"sigma","status":"experimental","id":"11111111-1111-1111-1111-111111111111","logsource":{"product":"windows","service":"security"}}
`

const rawAggregate = `{"group":"Sigma","kind":"aggregate","documents":[{"kind":"evtx","path":"/evidence/sample.evtx","data":{"Event":{"System":{"EventID":4699}}}},{"kind":"evtx","path":"/evidence/sample.evtx","data":{"Event":{"System":{"EventID":4698}}}}],"name":"Test Aggregate Rule","timestamp":"2019-03-19T00:02:04.319945+00:00","authors":["unknown"],"level":"low","source":"sigma","status":"experimental","id":"22222222-2222-2222-2222-222222222222","logsource":{"product":"windows","service":"security"}}
`

func TestRewriteIndividual(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "raw.jsonl")
	dst := filepath.Join(dir, "dst.jsonl")
	require.NoError(t, os.WriteFile(src, []byte(rawIndividual), 0o644))
	require.NoError(t, rewrite(src, dst))

	data, err := os.ReadFile(dst)
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(data, &rec))

	assert.Equal(t, "Test Scheduled Task Created", rec["message"])
	assert.Equal(t, "2019-03-19T00:02:04.319945+00:00", rec["datetime"])
	assert.Equal(t, "Event time", rec["timestamp_desc"])
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", rec["id"])
	// the matched event is preserved verbatim under "document"
	assert.Contains(t, rec, "document")
	assert.NotContains(t, rec, "timestamp")
}

func TestRewriteAggregate(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "raw.jsonl")
	dst := filepath.Join(dir, "dst.jsonl")
	require.NoError(t, os.WriteFile(src, []byte(rawAggregate), 0o644))
	require.NoError(t, rewrite(src, dst))

	data, err := os.ReadFile(dst)
	require.NoError(t, err)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(data, &rec))

	assert.Equal(t, "Test Aggregate Rule", rec["message"])
	assert.Equal(t, "2019-03-19T00:02:04.319945+00:00", rec["datetime"])
	assert.Equal(t, "Event time", rec["timestamp_desc"])
	// aggregate hits carry "documents" (plural) instead of "document"
	assert.Contains(t, rec, "documents")
	assert.NotContains(t, rec, "document")
	docs, ok := rec["documents"].([]any)
	require.True(t, ok)
	assert.Len(t, docs, 2)
}

func TestRewriteMultipleLines(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "raw.jsonl")
	dst := filepath.Join(dir, "dst.jsonl")
	// a blank line in the middle mirrors a stray trailing newline in real --jsonl output;
	// it must be skipped rather than fail the whole rewrite
	require.NoError(t, os.WriteFile(src, []byte(rawIndividual+"\n"+rawAggregate), 0o644))
	require.NoError(t, rewrite(src, dst))

	f, err := os.Open(dst)
	require.NoError(t, err)
	defer f.Close()

	var messages []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var rec map[string]any
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &rec))
		messages = append(messages, rec["message"].(string))
	}
	require.NoError(t, scanner.Err())

	assert.Equal(t, []string{"Test Scheduled Task Created", "Test Aggregate Rule"}, messages)
}
