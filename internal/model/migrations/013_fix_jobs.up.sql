CREATE TABLE IF NOT EXISTS jobs2 (
    id           TEXT NOT NULL PRIMARY KEY,
	evidence_id  TEXT NOT NULL,
	case_id      TEXT NOT NULL,
	name         TEXT NOT NULL,
	status       TEXT NOT NULL,
	error        TEXT NOT NULL,
	server_token TEXT NOT NULL,
	worker_token TEXT NOT NULL,

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
	FOREIGN KEY (evidence_id) REFERENCES evidences(id) ON DELETE CASCADE ON UPDATE CASCADE
);

INSERT INTO jobs2 (id, evidence_id, case_id, name, status, error, server_token, worker_token)
SELECT id, evidence_id, case_id, name, status, error, server_token, worker_token
FROM jobs;

DROP TABLE jobs;
ALTER TABLE jobs2 RENAME TO jobs;