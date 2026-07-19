CREATE INDEX event_assets_asset_id ON event_assets (asset_id);
CREATE INDEX event_indicators_indicator_id ON event_indicators (indicator_id);
CREATE INDEX indicators_value_type_nocase ON indicators (value COLLATE NOCASE, type);
CREATE INDEX jobs_object_id ON jobs (object_id);
