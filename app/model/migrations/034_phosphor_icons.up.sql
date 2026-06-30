-- Remap default enum icons from Heroicons (hio-*) to Phosphor (ph-*) names.
-- Only rows still holding a seeded default are rewritten; user-customized
-- icons are left untouched.
UPDATE enums SET icon='ph-archive' WHERE icon='hio-archive-box';
UPDATE enums SET icon='ph-download-simple' WHERE icon='hio-arrow-down-tray';
UPDATE enums SET icon='ph-arrows-clockwise' WHERE icon='hio-arrow-path';
UPDATE enums SET icon='ph-escalator-up' WHERE icon='hio-arrow-right-start-on-rectangle';
UPDATE enums SET icon='ph-arrows-left-right' WHERE icon='hio-arrows-right-left';
UPDATE enums SET icon='ph-flask' WHERE icon='hio-beaker';
UPDATE enums SET icon='ph-lightning' WHERE icon='hio-bolt';
UPDATE enums SET icon='ph-bug-beetle' WHERE icon='hio-bug-ant';
UPDATE enums SET icon='ph-bird' WHERE icon='hio-camera';
UPDATE enums SET icon='ph-check-circle' WHERE icon='hio-check-circle';
UPDATE enums SET icon='ph-clipboard-text' WHERE icon='hio-clipboard-document-check';
UPDATE enums SET icon='ph-gear-six' WHERE icon='hio-cog-6-tooth';
UPDATE enums SET icon='ph-terminal-window' WHERE icon='hio-command-line';
UPDATE enums SET icon='ph-desktop' WHERE icon='hio-computer-desktop';
UPDATE enums SET icon='ph-cpu' WHERE icon='hio-cpu-chip';
UPDATE enums SET icon='ph-cube' WHERE icon='hio-cube';
UPDATE enums SET icon='ph-file' WHERE icon='hio-document';
UPDATE enums SET icon='ph-file-text' WHERE icon='hio-document-text';
UPDATE enums SET icon='ph-eye' WHERE icon='hio-eye';
UPDATE enums SET icon='ph-eye-slash' WHERE icon='hio-eye-slash';
UPDATE enums SET icon='ph-fingerprint' WHERE icon='hio-finger-print';
UPDATE enums SET icon='ph-fire' WHERE icon='hio-fire';
UPDATE enums SET icon='ph-folder-open' WHERE icon='hio-folder-open';
UPDATE enums SET icon='ph-globe' WHERE icon='hio-globe-europe-africa';
UPDATE enums SET icon='ph-heart' WHERE icon='hio-heart';
UPDATE enums SET icon='ph-identification-card' WHERE icon='hio-identification';
UPDATE enums SET icon='ph-link' WHERE icon='hio-link';
UPDATE enums SET icon='ph-lock-open' WHERE icon='hio-lock-open';
UPDATE enums SET icon='ph-magnifying-glass' WHERE icon='hio-magnifying-glass';
UPDATE enums SET icon='ph-map-pin' WHERE icon='hio-map-pin';
UPDATE enums SET icon='ph-play' WHERE icon='hio-play';
UPDATE enums SET icon='ph-question' WHERE icon='hio-question-mark-circle';
UPDATE enums SET icon='ph-hard-drives' WHERE icon='hio-server';
UPDATE enums SET icon='ph-cloud-arrow-up' WHERE icon='hio-truck';
UPDATE enums SET icon='ph-user' WHERE icon='hio-user';
