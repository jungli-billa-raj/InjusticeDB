-- Add soft-delete column to assets
ALTER TABLE assets ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;

-- Index soft-deleted assets for quick queries and background cleanup
CREATE INDEX idx_assets_deleted_at ON assets(deleted_at) WHERE deleted_at IS NOT NULL;