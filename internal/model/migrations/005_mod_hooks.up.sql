CREATE TABLE IF NOT EXISTS hooks (
	id        TEXT NOT NULL PRIMARY KEY,
	trigger   TEXT NOT NULL,
	name      TEXT NOT NULL,
	mod       TEXT NOT NULL,
	condition TEXT NOT NULL,
	enabled   BOOLEAN NOT NULL
);

INSERT INTO hooks (id, trigger, name, mod, condition, enabled) VALUES
	(hex(randomblob(5)), 'OnEvidenceAdded', 'Process donald archives with plaso --parsers win7', 'Plaso (Windows Preset)', 'evidence.Name endsWith ".donald.zip"', false),
	(hex(randomblob(5)), 'OnEvidenceAdded', 'Process donald archives with hayabusa', 'Hayabusa', 'evidence.Name endsWith ".donald.zip"', false),
	(hex(randomblob(5)), 'OnEvidenceAdded', 'Process evtx with hayabusa', 'Hayabusa', 'evidence.Name endsWith ".evtx"', false),
	(hex(randomblob(5)), 'OnEvidenceAdded', 'Ingest plaso timeline into timesketch', 'Upload Timeline to Timesketch', 'evidence.Name endsWith ".plaso"', false),
	(hex(randomblob(5)), 'OnEvidenceAdded', 'Ingest hayabusa timeline into timesketch', 'Upload Timeline to Timesketch', 'evidence.Name endsWith ".hayabusa.jsonl"', false),
	(hex(randomblob(5)), 'OnEvidenceAdded', 'Ingest hayabusa timeline into dagobert', 'Ingest Hayabusa Timeline', 'evidence.Name endsWith ".hayabusa.jsonl"', false);