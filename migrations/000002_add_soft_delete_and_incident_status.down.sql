DROP INDEX IF EXISTS idx_assets_deleted_at;
ALTER TABLE assets DROP COLUMN IF EXISTS deleted_at;