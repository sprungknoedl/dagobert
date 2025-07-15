INSERT INTO policies (ptype, v0, v1, v2) VALUES ('p', '*', '/public/*', '*');
DELETE FROM policies WHERE v1 = '/web/*';