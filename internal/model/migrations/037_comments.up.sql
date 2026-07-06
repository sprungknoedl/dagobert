CREATE TABLE comments (
    id        TEXT PRIMARY KEY,
    case_id   TEXT NOT NULL,
    kind      TEXT NOT NULL,   -- events|assets|indicators|evidences|malware|tasks
    object_id TEXT NOT NULL,
    author    TEXT NOT NULL,   -- user UPN
    time      DATETIME NOT NULL,
    message   TEXT NOT NULL
);
CREATE INDEX comments_object ON comments (case_id, kind, object_id);
