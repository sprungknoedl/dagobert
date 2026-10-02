UPDATE automation_rules SET condition = replace(condition, 'evidence.Name', 'obj.Name')
WHERE condition LIKE 'evidence.Name endsWith %';

UPDATE automation_rules SET module = 'Plaso' WHERE module = 'Plaso (Windows Preset)';
UPDATE automation_rules SET module = 'Timesketch Importer' WHERE module = 'Upload Timeline to Timesketch';
UPDATE automation_rules SET module = 'Alert Importer' WHERE module = 'Ingest Hayabusa Timeline';
