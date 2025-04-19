ALTER TABLE cases ADD COLUMN summary TEXT NOT NULL DEFAULT '';
UPDATE cases SET summary = summary_who || x'0a0a'
    || summary_what || x'0a0a'
    || summary_when || x'0a0a'
    || summary_where || x'0a0a'
    || summary_why || x'0a0a'
    || summary_how;

ALTER TABLE cases DROP COLUMN summary_who;
ALTER TABLE cases DROP COLUMN summary_what;
ALTER TABLE cases DROP COLUMN summary_when;
ALTER TABLE cases DROP COLUMN summary_where;
ALTER TABLE cases DROP COLUMN summary_why;
ALTER TABLE cases DROP COLUMN summary_how;
