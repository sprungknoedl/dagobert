-- Re-seed the KeyTypes enum rows (012-style) so the DB-defined set is restored.
INSERT INTO enums (id, category, rank, name, icon, state) VALUES
	(hex(randomblob(5)), 'KeyTypes', 0, 'API', 'hio-beaker', ''),
	(hex(randomblob(5)), 'KeyTypes', 0, 'Donald', 'hio-camera', '');

DELETE FROM policies WHERE ptype = 'p' AND v0 = 'role::Donald' AND v1 = '/cases/*/evidences/new' AND v2 = 'POST';
DELETE FROM policies WHERE ptype = 'g' AND v0 = '<donald>' AND v1 = 'role::Donald';
DELETE FROM users WHERE id = '<donald>';
