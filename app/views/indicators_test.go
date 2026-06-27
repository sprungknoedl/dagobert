package views

import (
	"testing"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummarizeEnrichment(t *testing.T) {
	t.Run("empty input returns no results and severity -1", func(t *testing.T) {
		worst, sev, results := summarizeEnrichment(nil)
		assert.Equal(t, "", worst)
		assert.Equal(t, -1, sev)
		assert.Empty(t, results)
	})

	t.Run("rows without a recognised verdict are ignored", func(t *testing.T) {
		worst, sev, results := summarizeEnrichment([]model.Enrichment{
			{Module: "VirusTotal", Summary: "lots of hits", Link: "https://virustotal.com/..."},
		})
		assert.Equal(t, "", worst)
		assert.Equal(t, -1, sev)
		assert.Empty(t, results)
	})

	t.Run("a row with a verdict is included, source is the module", func(t *testing.T) {
		_, _, results := summarizeEnrichment([]model.Enrichment{
			{Module: "VirusTotal", Verdict: "malicious"},
		})
		require.Len(t, results, 1)
		assert.Equal(t, "VirusTotal", results[0].Source)
		assert.Equal(t, "malicious", results[0].Verdict)
		assert.Equal(t, 3, results[0].Severity)
	})

	t.Run("worst verdict is selected across multiple sources", func(t *testing.T) {
		worst, sev, _ := summarizeEnrichment([]model.Enrichment{
			{Module: "MISP", Verdict: "clean"},
			{Module: "VirusTotal", Verdict: "malicious"},
			{Module: "Abuse", Verdict: "suspicious"},
		})
		assert.Equal(t, "malicious", worst)
		assert.Equal(t, 3, sev)
	})

	t.Run("results sorted alphabetically by source case-insensitive", func(t *testing.T) {
		_, _, results := summarizeEnrichment([]model.Enrichment{
			{Module: "VirusTotal", Verdict: "malicious"},
			{Module: "MISP", Verdict: "clean"},
			{Module: "Abuse", Verdict: "suspicious"},
		})
		require.Len(t, results, 3)
		assert.Equal(t, "Abuse", results[0].Source)
		assert.Equal(t, "MISP", results[1].Source)
		assert.Equal(t, "VirusTotal", results[2].Source)
	})

	t.Run("matching is case-insensitive and verdict canonicalised", func(t *testing.T) {
		worst, sev, results := summarizeEnrichment([]model.Enrichment{
			{Module: "Tool", Verdict: "Malicious"},
		})
		assert.Equal(t, "malicious", worst)
		assert.Equal(t, 3, sev)
		require.Len(t, results, 1)
		assert.Equal(t, "malicious", results[0].Verdict)
	})

	t.Run("descriptive rows with empty verdict are ignored", func(t *testing.T) {
		worst, sev, results := summarizeEnrichment([]model.Enrichment{
			{Module: "WHOIS", Summary: "registered 2001", Verdict: ""},
			{Module: "VirusTotal", Verdict: "suspicious"},
		})
		assert.Equal(t, "suspicious", worst)
		assert.Equal(t, 2, sev)
		assert.Len(t, results, 1)
	})
}
