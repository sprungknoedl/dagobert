ALTER TABLE hooks RENAME COLUMN module TO mod;

DROP TABLE jobs;
CREATE TABLE IF NOT EXISTS jobs (
	id           TEXT NOT NULL PRIMARY KEY,
	case_id      TEXT NOT NULL,
	evidence_id  TEXT NOT NULL,
	name         TEXT NOT NULL,
	status       TEXT NOT NULL,
	error        TEXT NOT NULL,
	server_token TEXT NOT NULL,
	worker_token TEXT NOT NULL,

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
	FOREIGN KEY (evidence_id) REFERENCES evidences(id) ON DELETE CASCADE ON UPDATE CASCADE
);