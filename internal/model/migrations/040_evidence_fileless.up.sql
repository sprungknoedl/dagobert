ALTER TABLE evidences ADD COLUMN fileless BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE evidences SET fileless = (size = 0);
CREATE UNIQUE INDEX ux_evidences_case_name ON evidences (case_id, name);
