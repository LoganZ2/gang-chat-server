ALTER TABLE assets ADD COLUMN storage_key TEXT;

UPDATE assets
SET storage_key = 'assets/' || id || '/' || filename
WHERE storage_key IS NULL OR storage_key = '';

CREATE INDEX IF NOT EXISTS idx_assets_storage_key ON assets(storage_key);
