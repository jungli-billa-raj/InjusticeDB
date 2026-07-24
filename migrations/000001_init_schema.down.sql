-- Disable RLS Policies
DROP POLICY IF EXISTS message_participant_policy ON messages;
DROP POLICY IF EXISTS conversation_participant_policy ON conversations;

-- Drop Indexes
DROP INDEX IF EXISTS idx_incidents_verification; 
DROP INDEX IF EXISTS idx_revisions_incident ;
DROP INDEX IF EXISTS idx_assets_incident_id ;
DROP INDEX IF EXISTS idx_comments_incident_id; 
DROP INDEX IF EXISTS idx_incident_culprits_person_id ;
DROP INDEX IF EXISTS idx_conversations_users ;
DROP INDEX IF EXISTS idx_messages_conversation;
DROP INDEX IF EXISTS idx_assets_incident_active;

-- Drop Triggers and Functions
DROP TRIGGER IF EXISTS update_incidents_updated_at ON incidents;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop Tables
DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS conversations CASCADE;
DROP TABLE IF EXISTS ydcidc_targets CASCADE;
DROP TABLE IF EXISTS comments CASCADE;
DROP TABLE IF EXISTS assets CASCADE;
DROP TABLE IF EXISTS incident_verifications CASCADE;
DROP TABLE IF EXISTS incident_culprits CASCADE;
DROP TABLE IF EXISTS incident_revisions CASCADE;
DROP TABLE IF EXISTS incidents CASCADE;
DROP TABLE IF EXISTS people CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Drop Custom ENUM Types
DROP TYPE IF EXISTS vote_type;
DROP TYPE IF EXISTS asset_type;
DROP TYPE IF EXISTS culprit_status_type;
DROP TYPE IF EXISTS justice_status_type;
DROP TYPE IF EXISTS verification_status_type;
DROP TYPE IF EXISTS user_role;