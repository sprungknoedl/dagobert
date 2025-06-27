INSERT INTO enums (id, category, rank, name, icon, state) VALUES
	(hex(randomblob(5)), 'KeyTypes',  0, 'API', 'hio-beaker', ''),
	(hex(randomblob(5)), 'KeyTypes',  0, 'Dagobert', 'hio-bolt', ''),
	(hex(randomblob(5)), 'KeyTypes',  0, 'Donald', 'hio-camera', '');

UPDATE enums SET rank = 0 WHERE category='IndicatorTLPs' and name='TLP:RED';
UPDATE enums SET rank = 1 WHERE category='IndicatorTLPs' and name='TLP:AMBER';
UPDATE enums SET rank = 2 WHERE category='IndicatorTLPs' and name='TLP:GREEN';
UPDATE enums SET rank = 3 WHERE category='IndicatorTLPs' and name='TLP:CLEAR';