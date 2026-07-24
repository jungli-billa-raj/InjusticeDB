-- Drop the index created for parent_id lookups
DROP INDEX IF EXISTS idx_comments_parent_id;

-- Remove the parent_id column from the comments table
ALTER TABLE comments DROP COLUMN IF EXISTS parent_id;