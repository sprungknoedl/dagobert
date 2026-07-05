ALTER TABLE users ADD COLUMN password TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN provider TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN access_token TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN refresh_token TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN expiry DATETIME NOT NULL DEFAULT '1970-01-01T00:00:00Z';

INSERT INTO users (id, upn, name, email, role) VALUES ('<system>', '<system>', 'System', 'system@dagobert', 'Administrator');