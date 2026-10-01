CREATE TABLE IF NOT EXISTS tasks2 (
	id       TEXT NOT NULL PRIMARY KEY,
	case_id  TEXT NOT NULL,
	type     TEXT NOT NULL,
	task     TEXT NOT NULL,
	done     BOOLEAN NOT NULL,
	owner_id TEXT,
	date_due DATETIME NOT NULL,
	custom   TEXT NOT NULL DEFAULT '',

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE,
	FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE
);

INSERT INTO tasks2 (id, case_id, type, task, done, owner_id, date_due, custom)
SELECT
	t.id, t.case_id, t.type, t.task, t.done,
	(SELECT u.id FROM users u
	 WHERE LOWER(u.login) = LOWER(t.owner) OR LOWER(u.name) = LOWER(t.owner) OR LOWER(u.email) = LOWER(t.owner)
	 LIMIT 1),
	t.date_due, t.custom
FROM tasks t;

DROP TABLE tasks;
ALTER TABLE tasks2 RENAME TO tasks;
