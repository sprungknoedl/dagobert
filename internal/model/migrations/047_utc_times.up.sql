-- Rewrite stored times to UTC with a 9-digit fraction (model.Time's timeLayout),
-- so that text ordering is chronological. Done by hand rather than with
-- strftime('%f') because SQLite only keeps milliseconds.

UPDATE events SET time =
	strftime('%Y-%m-%dT%H:%M:%S', substr(time, 1, 19) || CASE WHEN time LIKE '%Z' THEN 'Z' ELSE substr(time, -6) END) || '.' ||
	substr(CASE WHEN substr(time, 20, 1) = '.' THEN substr(time, 21, length(time) - 20 - length(CASE WHEN time LIKE '%Z' THEN 'Z' ELSE substr(time, -6) END)) ELSE '' END || '000000000', 1, 9) || 'Z'
WHERE time LIKE '____-__-__T__:__:__%';

UPDATE comments SET time =
	strftime('%Y-%m-%dT%H:%M:%S', substr(time, 1, 19) || CASE WHEN time LIKE '%Z' THEN 'Z' ELSE substr(time, -6) END) || '.' ||
	substr(CASE WHEN substr(time, 20, 1) = '.' THEN substr(time, 21, length(time) - 20 - length(CASE WHEN time LIKE '%Z' THEN 'Z' ELSE substr(time, -6) END)) ELSE '' END || '000000000', 1, 9) || 'Z'
WHERE time LIKE '____-__-__T__:__:__%';

UPDATE evidence_logs SET time =
	strftime('%Y-%m-%dT%H:%M:%S', substr(time, 1, 19) || CASE WHEN time LIKE '%Z' THEN 'Z' ELSE substr(time, -6) END) || '.' ||
	substr(CASE WHEN substr(time, 20, 1) = '.' THEN substr(time, 21, length(time) - 20 - length(CASE WHEN time LIKE '%Z' THEN 'Z' ELSE substr(time, -6) END)) ELSE '' END || '000000000', 1, 9) || 'Z'
WHERE time LIKE '____-__-__T__:__:__%';

UPDATE tasks SET date_due =
	strftime('%Y-%m-%dT%H:%M:%S', substr(date_due, 1, 19) || CASE WHEN date_due LIKE '%Z' THEN 'Z' ELSE substr(date_due, -6) END) || '.' ||
	substr(CASE WHEN substr(date_due, 20, 1) = '.' THEN substr(date_due, 21, length(date_due) - 20 - length(CASE WHEN date_due LIKE '%Z' THEN 'Z' ELSE substr(date_due, -6) END)) ELSE '' END || '000000000', 1, 9) || 'Z'
WHERE date_due LIKE '____-__-__T__:__:__%';

UPDATE users SET last_login =
	strftime('%Y-%m-%dT%H:%M:%S', substr(last_login, 1, 19) || CASE WHEN last_login LIKE '%Z' THEN 'Z' ELSE substr(last_login, -6) END) || '.' ||
	substr(CASE WHEN substr(last_login, 20, 1) = '.' THEN substr(last_login, 21, length(last_login) - 20 - length(CASE WHEN last_login LIKE '%Z' THEN 'Z' ELSE substr(last_login, -6) END)) ELSE '' END || '000000000', 1, 9) || 'Z'
WHERE last_login LIKE '____-__-__T__:__:__%';

UPDATE enrichments SET fetched_at =
	strftime('%Y-%m-%dT%H:%M:%S', substr(fetched_at, 1, 19) || CASE WHEN fetched_at LIKE '%Z' THEN 'Z' ELSE substr(fetched_at, -6) END) || '.' ||
	substr(CASE WHEN substr(fetched_at, 20, 1) = '.' THEN substr(fetched_at, 21, length(fetched_at) - 20 - length(CASE WHEN fetched_at LIKE '%Z' THEN 'Z' ELSE substr(fetched_at, -6) END)) ELSE '' END || '000000000', 1, 9) || 'Z'
WHERE fetched_at LIKE '____-__-__T__:__:__%';

UPDATE evidences SET starts_at =
	strftime('%Y-%m-%dT%H:%M:%S', substr(starts_at, 1, 19) || CASE WHEN starts_at LIKE '%Z' THEN 'Z' ELSE substr(starts_at, -6) END) || '.' ||
	substr(CASE WHEN substr(starts_at, 20, 1) = '.' THEN substr(starts_at, 21, length(starts_at) - 20 - length(CASE WHEN starts_at LIKE '%Z' THEN 'Z' ELSE substr(starts_at, -6) END)) ELSE '' END || '000000000', 1, 9) || 'Z'
WHERE starts_at LIKE '____-__-__T__:__:__%';

UPDATE evidences SET ends_at =
	strftime('%Y-%m-%dT%H:%M:%S', substr(ends_at, 1, 19) || CASE WHEN ends_at LIKE '%Z' THEN 'Z' ELSE substr(ends_at, -6) END) || '.' ||
	substr(CASE WHEN substr(ends_at, 20, 1) = '.' THEN substr(ends_at, 21, length(ends_at) - 20 - length(CASE WHEN ends_at LIKE '%Z' THEN 'Z' ELSE substr(ends_at, -6) END)) ELSE '' END || '000000000', 1, 9) || 'Z'
WHERE ends_at LIKE '____-__-__T__:__:__%';
