package views

import (
	"testing"

	"github.com/sprungknoedl/dagobert/app/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummarizeEnrichment(t *testing.T) {
	t.Run("empty custom returns no results and severity -1", func(t *testing.T) {
		worst, sev, results := summarizeEnrichment(model.Custom{})
		assert.Equal(t, "", worst)
		assert.Equal(t, -1, sev)
		assert.Empty(t, results)
	})

	t.Run("out-of-set values are ignored", func(t *testing.T) {
		worst, sev, results := summarizeEnrichment(model.Custom{
			"VirusTotal Enrichment": "lots of hits",
			"VirusTotal Link":       "https://virustotal.com/...",
		})
		assert.Equal(t, "", worst)
		assert.Equal(t, -1, sev)
		assert.Empty(t, results)
	})

	t.Run("in-set values are included and source suffix stripped", func(t *testing.T) {
		_, _, results := summarizeEnrichment(model.Custom{
			"VirusTotal Verdict": "malicious",
		})
		require.Len(t, results, 1)
		assert.Equal(t, "VirusTotal", results[0].Source)
		assert.Equal(t, "malicious", results[0].Verdict)
		assert.Equal(t, 3, results[0].Severity)
	})

	t.Run("key without Verdict suffix kept verbatim", func(t *testing.T) {
		_, _, results := summarizeEnrichment(model.Custom{
			"Analyst call": "clean",
		})
		require.Len(t, results, 1)
		assert.Equal(t, "Analyst call", results[0].Source)
	})

	t.Run("worst verdict is selected across multiple sources", func(t *testing.T) {
		worst, sev, _ := summarizeEnrichment(model.Custom{
			"MISP Verdict":       "clean",
			"VirusTotal Verdict": "malicious",
			"Abuse Verdict":      "suspicious",
		})
		assert.Equal(t, "malicious", worst)
		assert.Equal(t, 3, sev)
	})

	t.Run("results sorted alphabetically by source case-insensitive", func(t *testing.T) {
		_, _, results := summarizeEnrichment(model.Custom{
			"VirusTotal Verdict": "malicious",
			"MISP Verdict":       "clean",
			"Abuse Verdict":      "suspicious",
		})
		require.Len(t, results, 3)
		assert.Equal(t, "Abuse", results[0].Source)
		assert.Equal(t, "MISP", results[1].Source)
		assert.Equal(t, "VirusTotal", results[2].Source)
	})

	t.Run("matching is case-insensitive and verdict canonicalised", func(t *testing.T) {
		worst, sev, results := summarizeEnrichment(model.Custom{
			"Tool Verdict": "Malicious",
		})
		assert.Equal(t, "malicious", worst)
		assert.Equal(t, 3, sev)
		require.Len(t, results, 1)
		assert.Equal(t, "malicious", results[0].Verdict)
	})

	t.Run("mixed in-set and out-of-set values", func(t *testing.T) {
		worst, _, results := summarizeEnrichment(model.Custom{
			"VirusTotal Verdict":    "suspicious",
			"VirusTotal Enrichment": "summary text",
			"VirusTotal Link":       "https://vt.com",
		})
		assert.Equal(t, "suspicious", worst)
		assert.Len(t, results, 1)
	})
}
