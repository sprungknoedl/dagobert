CREATE TABLE IF NOT EXISTS enums (
	id       TEXT NOT NULL PRIMARY KEY,
	category TEXT NOT NULL,
	rank     INTEGER NOT NULL,
	name     TEXT NOT NULL,
	icon     TEXT NOT NULL,
	state    TEXT NOT NULL
);

INSERT INTO enums (id, category, rank, name, icon, state) VALUES
	(hex(randomblob(5)), 'AssetStatus', -1, 'Under investigation', '', ''),
	(hex(randomblob(5)), 'AssetStatus',  0, 'Compromised', 'hio-bug-ant', 'error'),
	(hex(randomblob(5)), 'AssetStatus',  0, 'Accessed', 'hio-command-line', 'warning'),
	(hex(randomblob(5)), 'AssetStatus',  0, 'No sign of compromise', 'hio-check-circle', 'success'),
	(hex(randomblob(5)), 'AssetStatus',  0, 'Out of scope', '', ''),

	(hex(randomblob(5)), 'AssetTypes',  0, 'Account', 'hio-user', ''),
	(hex(randomblob(5)), 'AssetTypes',  0, 'Desktop', 'hio-computer-desktop', ''),
	(hex(randomblob(5)), 'AssetTypes',  0, 'Server', 'hio-server', ''),
	(hex(randomblob(5)), 'AssetTypes', 99, 'Other', 'hio-question-mark-circle', ''),

	(hex(randomblob(5)), 'CaseSeverities',  0, 'Low', '', ''),
	(hex(randomblob(5)), 'CaseSeverities',  0, 'Medium', '', ''),
	(hex(randomblob(5)), 'CaseSeverities',  0, 'High', '', ''),

	(hex(randomblob(5)), 'CaseOutcomes', -1, '', '', ''),
	(hex(randomblob(5)), 'CaseOutcomes',  0, 'False positive', '', ''),
	(hex(randomblob(5)), 'CaseOutcomes',  0, 'True positive', '', ''),
	(hex(randomblob(5)), 'CaseOutcomes',  0, 'Benign positive', '', ''),

	(hex(randomblob(5)), 'EventTypes',  0, 'C2', 'hio-server', ''),
	(hex(randomblob(5)), 'EventTypes',  0, 'Collection', 'hio-arrow-down-tray', ''),
	(hex(randomblob(5)), 'EventTypes',  0, 'Credential Access', 'hio-identification', ''),
	(hex(randomblob(5)), 'EventTypes',  0, 'Defense Evasion', 'hio-eye-slash', ''),
	(hex(randomblob(5)), 'EventTypes',  0, 'Discovery', 'hio-eye', ''),
	(hex(randomblob(5)), 'EventTypes',  0, 'Execution', 'hio-play', ''),
	(hex(randomblob(5)), 'EventTypes',  0, 'Exfiltration', 'hio-truck', ''),
	(hex(randomblob(5)), 'EventTypes',  0, 'Impact', 'hio-fire', ''),
	(hex(randomblob(5)), 'EventTypes',  0, 'Initial Access', 'hio-lock-open', ''),
	(hex(randomblob(5)), 'EventTypes',  0, 'Lateral Movement', 'hio-arrows-right-left', ''),
	(hex(randomblob(5)), 'EventTypes',  0, 'Persistence', 'hio-arrow-path', ''),
	(hex(randomblob(5)), 'EventTypes',  0, 'Privilege Escalation', 'hio-arrow-right-start-on-rectangle', ''),
	(hex(randomblob(5)), 'EventTypes',  0, 'Reconnaissance', 'hio-magnifying-glass', ''),
	(hex(randomblob(5)), 'EventTypes',  0, 'Resource Development', 'hio-cog-6-tooth', ''),
	(hex(randomblob(5)), 'EventTypes',  2, 'Legitimate', 'hio-check-circle', 'success'),
	(hex(randomblob(5)), 'EventTypes',  2, 'Remediation', 'hio-heart', 'success'),
	(hex(randomblob(5)), 'EventTypes', 99, 'Other', '', ''),

	(hex(randomblob(5)), 'EvidenceTypes',  0, 'File', 'hio-document', ''),
	(hex(randomblob(5)), 'EvidenceTypes',  0, 'Logs', 'hio-document-text', ''),
	(hex(randomblob(5)), 'EvidenceTypes',  0, 'Triage', 'hio-archive-box', ''),
	(hex(randomblob(5)), 'EvidenceTypes',  0, 'System Image', 'hio-server', ''),
	(hex(randomblob(5)), 'EvidenceTypes',  0, 'Memory Dump', 'hio-cpu-chip', ''),
	(hex(randomblob(5)), 'EvidenceTypes',  0, 'Malware', 'hio-bug-ant', ''),
	(hex(randomblob(5)), 'EvidenceTypes', 99, 'Other', 'hio-cube', ''),

	(hex(randomblob(5)), 'IndicatorStatus', -1, 'Under investigation', '', ''),
	(hex(randomblob(5)), 'IndicatorStatus',  0, 'Confirmed', 'hio-bug-ant', 'error'),
	(hex(randomblob(5)), 'IndicatorStatus',  0, 'Suspicious', 'hio-finger-print', 'warning'),
	(hex(randomblob(5)), 'IndicatorStatus',  0, 'Unrelated', 'hio-check-circle', 'success'),

	(hex(randomblob(5)), 'IndicatorTypes',  0, 'Domain', 'hio-globe-europe-africa', ''),
	(hex(randomblob(5)), 'IndicatorTypes',  0, 'Hash', 'hio-finger-print', ''),
	(hex(randomblob(5)), 'IndicatorTypes',  0, 'IP', 'hio-map-pin', ''),
	(hex(randomblob(5)), 'IndicatorTypes',  0, 'Path', 'hio-folder-open', ''),
	(hex(randomblob(5)), 'IndicatorTypes',  0, 'Service', 'hio-command-line', ''),
	(hex(randomblob(5)), 'IndicatorTypes',  0, 'URL', 'hio-link', ''),
	(hex(randomblob(5)), 'IndicatorTypes', 99, 'Other', 'hio-question-mark-circle', ''),

	(hex(randomblob(5)), 'IndicatorTLPs',  0, 'TLP:RED', '', 'error'),
	(hex(randomblob(5)), 'IndicatorTLPs',  0, 'TLP:AMBER', '', 'warning'),
	(hex(randomblob(5)), 'IndicatorTLPs',  0, 'TLP:GREEN', '', 'success'),
	(hex(randomblob(5)), 'IndicatorTLPs',  0, 'TLP:CLEAR', '', ''),

	(hex(randomblob(5)), 'MalwareStatus', -1, 'Under investigation', '', ''),
	(hex(randomblob(5)), 'MalwareStatus',  0, 'Malicious', 'hio-bug-ant', 'error'),
	(hex(randomblob(5)), 'MalwareStatus',  0, 'Suspicious', 'hio-finger-print', 'warning'),
	(hex(randomblob(5)), 'MalwareStatus',  0, 'Unrelated', 'hio-check-circle', 'success'),

	(hex(randomblob(5)), 'TaskTypes',  0, 'Information request', 'hio-question-mark-circle', ''),
	(hex(randomblob(5)), 'TaskTypes',  0, 'Analysis', 'hio-magnifying-glass', ''),
	(hex(randomblob(5)), 'TaskTypes',  0, 'Deliverable', 'hio-document-text', ''),
	(hex(randomblob(5)), 'TaskTypes',  0, 'Checkpoint', 'hio-clipboard-document-check', ''),
	(hex(randomblob(5)), 'TaskTypes', 99, 'Other', 'hio-question-mark-circle', '');