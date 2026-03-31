ALTER TABLE cases ADD COLUMN summary TEXT NOT NULL DEFAULT '';
UPDATE cases SET summary = TRIM(
    summary_who || CASE WHEN summary_who != '' THEN x'0a0a' ELSE '' END ||
    summary_what || CASE WHEN summary_what != '' THEN x'0a0a' ELSE '' END ||
    summary_when || CASE WHEN summary_when != '' THEN x'0a0a' ELSE '' END ||
    summary_where || CASE WHEN summary_where != '' THEN x'0a0a' ELSE '' END ||
    summary_why || CASE WHEN summary_why != '' THEN x'0a0a' ELSE '' END ||
    summary_how
);
ALTER TABLE cases DROP COLUMN summary_who;
ALTER TABLE cases DROP COLUMN summary_what;
ALTER TABLE cases DROP COLUMN summary_when;
ALTER TABLE cases DROP COLUMN summary_where;
ALTER TABLE cases DROP COLUMN summary_why;
ALTER TABLE cases DROP COLUMN summary_how;

ALTER TABLE indicators ADD COLUMN flagged BOOLEAN NOT NULL DEFAULT FALSE;
