ALTER TABLE cases DROP COLUMN custom;
ALTER TABLE assets DROP COLUMN custom;
ALTER TABLE events DROP COLUMN custom;
ALTER TABLE evidences DROP COLUMN custom;
ALTER TABLE indicators DROP COLUMN custom;
ALTER TABLE malware DROP COLUMN custom;
ALTER TABLE notes DROP COLUMN custom;
ALTER TABLE tasks DROP COLUMN custom;

DROP TABLE IF EXISTS custom_attributes;
