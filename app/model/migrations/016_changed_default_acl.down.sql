DELETE FROM policies WHERE v1 == "/" AND v0 == "*";
INSERT INTO policies (ptype, v0, v1, v2) VALUES 
	('p', 'role::User', '/', 'GET'),
	('p', 'role::Read-Only', '/', 'GET'),
	ON CONFLICT DO NOTHING;