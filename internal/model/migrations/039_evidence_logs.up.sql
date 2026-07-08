CREATE TABLE evidence_logs (
    id          TEXT PRIMARY KEY,
    case_id     TEXT NOT NULL,
    evidence_id TEXT NOT NULL,
    name        TEXT NOT NULL,
    user        TEXT NOT NULL,
    event       TEXT NOT NULL,   -- uploaded|downloaded|edited|module run|deleted
    details     TEXT NOT NULL,
    time        DATETIME NOT NULL
);
CREATE INDEX evidence_logs_evidence ON evidence_logs (case_id, evidence_id);
