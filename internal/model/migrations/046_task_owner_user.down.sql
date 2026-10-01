CREATE TABLE IF NOT EXISTS tasks2 (
	id       TEXT NOT NULL PRIMARY KEY,
	case_id  TEXT NOT NULL,
	type     TEXT NOT NULL,
	task     TEXT NOT NULL,
	done     BOOLEAN NOT NULL,
	owner    TEXT NOT NULL DEFAULT '',
	date_due DATETIME NOT NULL,
	custom   TEXT NOT NULL DEFAULT '',

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

INSERT INTO tasks2 (id, case_id, type, task, done, owner, date_due, custom)
SELECT t.id, t.case_id, t.type, t.task, t.done, COALESCE(u.name, ''), t.date_due, t.custom
FROM tasks t
LEFT JOIN users u ON u.id = t.owner_id;

DROP TABLE tasks;
ALTER TABLE tasks2 RENAME TO tasks;
