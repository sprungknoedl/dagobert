CREATE INDEX events_case_id ON events (case_id, time);
CREATE INDEX jobs_status ON jobs (status);
CREATE INDEX enrichments_case_id ON enrichments (case_id, object_type);
CREATE INDEX assets_case_id ON assets (case_id, name);
