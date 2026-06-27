CREATE TABLE IF NOT EXISTS enrichments (
	id          TEXT NOT NULL PRIMARY KEY,
	case_id     TEXT NOT NULL DEFAULT '',
	object_type TEXT NOT NULL DEFAULT '',
	object_id   TEXT NOT NULL DEFAULT '',
	module      TEXT NOT NULL DEFAULT '',
	verdict     TEXT NOT NULL DEFAULT '',
	summary     TEXT NOT NULL DEFAULT '',
	link        TEXT NOT NULL DEFAULT '',
	fetched_at  DATETIME,

	UNIQUE (object_type, object_id, module)
);

-- Enrichment now lives in its own table; the custom-attribute system reverts to
-- analyst-defined fields only. Drop the stale enrichment definitions so they
-- stop rendering as empty inputs on the indicator form.
DELETE FROM custom_attributes WHERE entity = 'Indicator' AND label IN (
	'VirusTotal Verdict','VirusTotal Enrichment','VirusTotal Link',
	'AbuseIPDB Verdict','AbuseIPDB Enrichment','AbuseIPDB Link',
	'Hybrid Analysis Verdict','Hybrid Analysis Enrichment','Hybrid Analysis Link');

-- Strip the stale enrichment values out of indicators.custom (json_remove
-- ignores missing paths, so this is safe on every row). No backfill into the
-- new table — workers repopulate on next run.
UPDATE indicators SET custom = json_remove(custom,
	'$."VirusTotal Verdict"','$."VirusTotal Enrichment"','$."VirusTotal Link"',
	'$."AbuseIPDB Verdict"','$."AbuseIPDB Enrichment"','$."AbuseIPDB Link"',
	'$."Hybrid Analysis Verdict"','$."Hybrid Analysis Enrichment"','$."Hybrid Analysis Link"')
WHERE custom IS NOT NULL AND custom != '';
