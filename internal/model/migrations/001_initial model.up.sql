PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;

CREATE TABLE IF NOT EXISTS auditlog (
	time     DATETIME NOT NULL,
	user     TEXT NOT NULL,
	kase     TEXT NOT NULL,
	object   TEXT NOT NULL,
	activity TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS cases (
	id             TEXT NOT NULL PRIMARY KEY,
	name           TEXT NOT NULL,
	closed         BOOLEAN NOT NULL,
	classification TEXT NOT NULL,
	severity       TEXT NOT NULL,
	outcome        TEXT NOT NULL,
	summary        TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS assets (
	id          TEXT NOT NULL PRIMARY KEY,
	case_id     TEXT NOT NULL,
	status      TEXT NOT NULL,
	type        TEXT NOT NULL,
	name        TEXT NOT NULL,
	addr        TEXT NOT NULL,
	notes       TEXT NOT NULL,

	UNIQUE (name, case_id),
	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS events (
	id      TEXT NOT NULL PRIMARY KEY,
	case_id TEXT NOT NULL,
	time    DATETIME NOT NULL,
	type    TEXT NOT NULL,
	event   TEXT NOT NULL,
	raw     TEXT NOT NULL,
	flagged BOOLEAN NOT NULL,

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS event_assets (
	event_id TEXT NOT NULL,
	asset_id TEXT NOT NULL,

	FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE ON UPDATE CASCADE,
	FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS event_indicators (
	event_id     TEXT NOT NULL,
	indicator_id TEXT NOT NULL,

	FOREIGN KEY (indicator_id) REFERENCES indicators(id) ON DELETE CASCADE ON UPDATE CASCADE,
	FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS evidences (
	id       TEXT NOT NULL PRIMARY KEY,
	case_id  TEXT NOT NULL,
	name     TEXT NOT NULL,
	type     TEXT NOT NULL,
	size     INTEGER NOT NULL,
	source   TEXT NOT NULL,
	notes    TEXT NOT NULL,
	hash     TEXT NOT NULL,
	location TEXT NOT NULL,

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS reports (
	id       TEXT NOT NULL PRIMARY KEY,
	name     TEXT NOT NULL,
	notes    TEXT NOT NULL,

	UNIQUE (name)
);

CREATE TABLE IF NOT EXISTS runs (
	evidence_id TEXT NOT NULL,
	name        TEXT NOT NULL,
	description TEXT NOT NULL,
	status      TEXT NOT NULL,
	error       TEXT NOT NULL,
	ttl         DATETIME NOT NULL,

	PRIMARY KEY (evidence_id, name),
	FOREIGN KEY (evidence_id) REFERENCES evidences(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS indicators (
	id          TEXT NOT NULL PRIMARY KEY,
	case_id     TEXT NOT NULL,
	status      TEXT NOT NULL,
	type        TEXT NOT NULL,
	value       TEXT NOT NULL,
	tlp         TEXT NOT NULL,
	notes       TEXT NOT NULL,
	source      TEXT NOT NULL,

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS keys (
	key  TEXT NOT NULL PRIMARY KEY,
	name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS malware (
	id       TEXT NOT NULL PRIMARY KEY,
	asset_id TEXT NOT NULL,
	case_id  TEXT NOT NULL,
	status   TEXT NOT NULL,
	path     TEXT NOT NULL,
	hash     TEXT NOT NULL,
	notes    TEXT NOT NULL,

	FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE ON UPDATE CASCADE
	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS notes (
	id          TEXT NOT NULL PRIMARY KEY,
	case_id     TEXT NOT NULL,
	title       TEXT NOT NULL,
	category    TEXT NOT NULL,
	description TEXT NOT NULL,

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS tasks (
	id       TEXT NOT NULL PRIMARY KEY,
	case_id  TEXT NOT NULL,
	type     TEXT NOT NULL,
	task     TEXT NOT NULL,
	done     BOOLEAN NOT NULL,
	owner    TEXT NOT NULL,
	date_due DATETIME NOT NULL,

	FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS users (
	id         TEXT NOT NULL PRIMARY KEY,
	name       TEXT NOT NULL,
	upn        TEXT NOT NULL,
	email      TEXT NOT NULL,
	role       TEXT NOT NULL DEFAULT 'Read-Only',
	last_login DATETIME,

	UNIQUE (upn)
);

CREATE TABLE IF NOT EXISTS policies (
	ptype  TEXT NOT NULL,
	v0     TEXT NOT NULL DEFAULT '',
	v1     TEXT NOT NULL DEFAULT '',
	v2     TEXT NOT NULL DEFAULT '',
	v3     TEXT NOT NULL DEFAULT '',
	v4     TEXT NOT NULL DEFAULT '',
	v5     TEXT NOT NULL DEFAULT '',

	UNIQUE(ptype,v0,v1,v2,v3,v4,v5)
);

INSERT INTO policies (ptype, v0, v1, v2) VALUES 
	('p', '*', '/auth/*', '*'),
	('p', '*', '/web/*', '*'),
	('p', 'role::User', '/', 'GET'),
	('p', 'role::User', '/cases/', 'GET'),
	('p', 'role::Read-Only', '/', 'GET'),
	('p', 'role::Read-Only', '/cases/', 'GET'),
	('p', 'role::Administrator', '*', '*')
	ON CONFLICT DO NOTHING;