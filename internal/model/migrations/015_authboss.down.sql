ALTER TABLE users DROP COLUMN password;
ALTER TABLE users DROP COLUMN provider;
ALTER TABLE users DROP COLUMN access_token;
ALTER TABLE users DROP COLUMN refresh_token;
ALTER TABLE users DROP COLUMN expiry;

DELETE FROM users WHERE id = '<system>';