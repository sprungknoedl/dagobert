ALTER TABLE hooks RENAME COLUMN mod IF EXISTS TO module;

DROP TABLE jobs;
CREATE TABLE IF NOT EXISTS jobs (
	id           TEXT NOT NULL PRIMARY KEY,
	case_id      TEXT NOT NULL,
	name         TEXT NOT NULL,
	status       TEXT NOT NULL,
	error        TEXT NOT NULL,
	results      TEXT,
	settings     TEXT,
	object_id    TEXT NOT NULL,
	object       TEXT NOT NULL,
	server_token TEXT NOT NULL,
	worker_token TEXT NOT NULL,

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);