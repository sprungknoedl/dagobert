UPDATE automation_rules SET condition = replace(condition, 'obj.Name', 'evidence.Name')
WHERE name IN (
	'Process donald archives with plaso --parsers win7',
	'Process donald archives with hayabusa',
	'Process evtx with hayabusa',
	'Ingest plaso timeline into timesketch',
	'Ingest hayabusa timeline into timesketch',
	'Ingest hayabusa timeline into dagobert'
);

UPDATE automation_rules SET module = 'Plaso (Windows Preset)' WHERE module = 'Plaso' AND name = 'Process donald archives with plaso --parsers win7';
UPDATE automation_rules SET module = 'Upload Timeline to Timesketch' WHERE module = 'Timesketch Importer' AND name IN ('Ingest plaso timeline into timesketch', 'Ingest hayabusa timeline into timesketch');
UPDATE automation_rules SET module = 'Ingest Hayabusa Timeline' WHERE module = 'Alert Importer' AND name = 'Ingest hayabusa timeline into dagobert';
