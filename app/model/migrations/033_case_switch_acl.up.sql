-- Grant the quick case-switch endpoint to the non-admin roles. Administrator
-- already matches via its '*,*' policy. The switcher itself only lists cases the
-- user can already access (it tests Allowed on each case's summary route).
INSERT INTO policies (ptype, v0, v1, v2) VALUES
	('p', 'role::User', '/cases/switch', 'GET'),
	('p', 'role::Read-Only', '/cases/switch', 'GET');
