CREATE TABLE IF NOT EXISTS auditlog (
	time     DATETIME NOT NULL,
	user     TEXT NOT NULL,
	kase     TEXT NOT NULL,
	object   TEXT NOT NULL,
	activity TEXT NOT NULL
);