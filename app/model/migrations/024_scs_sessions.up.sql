CREATE TABLE sessions (
    token  TEXT PRIMARY KEY,
    data   BLOB NOT NULL,
    expiry REAL NOT NULL
);
CREATE INDEX sessions_expiry_idx ON sessions(expiry);

-- The OAuth2 token columns existed only to satisfy authboss's OAuth2User
-- interface; nothing else reads them.
ALTER TABLE users DROP COLUMN provider;
ALTER TABLE users DROP COLUMN access_token;
ALTER TABLE users DROP COLUMN refresh_token;
ALTER TABLE users DROP COLUMN expiry;
