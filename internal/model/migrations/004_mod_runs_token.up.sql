ALTER TABLE runs DROP COLUMN ttl;
ALTER TABLE runs ADD COLUMN case_id TEXT NOT NULL DEFAULT '<invalid>';
ALTER TABLE runs ADD COLUMN token TEXT NOT NULL DEFAULT '<default>';