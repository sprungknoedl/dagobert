-- Bind the 'Donald' api key type to a least-privilege role. Donald keys now
-- authenticate as the shared '<donald>' principal, which may only create triage
-- evidence (POST /cases/*/evidences/new) and nothing else.
INSERT INTO users (id, upn, name, email, role) VALUES ('<donald>', '<donald>', 'Donald', 'donald@dagobert', 'Donald');
INSERT INTO policies (ptype, v0, v1) VALUES ('g', '<donald>', 'role::Donald');
INSERT INTO policies (ptype, v0, v1, v2) VALUES ('p', 'role::Donald', '/cases/*/evidences/new', 'POST');

-- Key types are now code-defined (see app/model/keys.go); drop the DB enum
-- category entirely. This supersedes the Dagobert-only delete in migration 026,
-- which still runs first and is harmless.
DELETE FROM enums WHERE category = 'KeyTypes';
