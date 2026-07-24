-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Define Custom Native ENUM Types
CREATE TYPE user_role AS ENUM ('user', 'moderator', 'admin');
CREATE TYPE verification_status_type AS ENUM ('pending', 'verified', 'rejected', 'disputed');
CREATE TYPE justice_status_type AS ENUM ('proceeding', 'served', 'stalled');
CREATE TYPE culprit_status_type AS ENUM ('suspect', 'accused', 'guilty', 'convicted');
CREATE TYPE asset_type AS ENUM ('image', 'video', 'article', 'archive_link');
CREATE TYPE vote_type AS ENUM ('verify', 'reject');

-- 1. USERS TABLE
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    auth_provider VARCHAR(50) NOT NULL DEFAULT 'email',
    role user_role NOT NULL DEFAULT 'user',
    credibility_score INTEGER NOT NULL DEFAULT 100 CHECK (credibility_score >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 2. PEOPLE TABLE
CREATE TABLE people (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(150) NOT NULL,
    organization VARCHAR(150),
    age INTEGER CHECK (age >= 0),
    state VARCHAR(100),
    city VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 3. INCIDENTS TABLE
CREATE TABLE incidents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    verification_status verification_status_type NOT NULL DEFAULT 'pending',
    current_version INTEGER NOT NULL DEFAULT 1,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 4. INCIDENT REVISIONS TABLE
CREATE TABLE incident_revisions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_id UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    full_story TEXT NOT NULL,
    severity INTEGER NOT NULL CHECK (severity >= 1 AND severity <= 10),
    justice_status justice_status_type NOT NULL DEFAULT 'proceeding',
    state VARCHAR(100) NOT NULL,
    city VARCHAR(100) NOT NULL,
    version_number INTEGER NOT NULL,
    change_summary TEXT NOT NULL,
    edited_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(incident_id, version_number)
);

-- 5. JUNCTION TABLE
CREATE TABLE incident_culprits (
    incident_id UUID REFERENCES incidents(id) ON DELETE CASCADE,
    person_id UUID REFERENCES people(id) ON DELETE CASCADE, 
    culprit_status culprit_status_type NOT NULL DEFAULT 'suspect',
    PRIMARY KEY (incident_id, person_id)
);

-- 6. INCIDENT VERIFICATIONS TABLE
CREATE TABLE incident_verifications (
    incident_id UUID REFERENCES incidents(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    vote vote_type NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (incident_id, user_id)
);

-- 7. ASSETS & SOURCES TABLE
CREATE TABLE assets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_id UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    type asset_type NOT NULL,
    url TEXT NOT NULL,
    archive_url TEXT,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ DEFAULT NULL
);

-- 8. COMMENTS TABLE
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_id UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 9. YDCIDC TARGETS TABLE
CREATE TABLE ydcidc_targets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(150) NOT NULL,
    occupation VARCHAR(100),
    state VARCHAR(100),
    city VARCHAR(100),
    cause_of_resentment TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 10. CONVERSATIONS TABLE
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_one_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_two_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_different_users CHECK (user_one_id <> user_two_id),
    CONSTRAINT check_canonical_order CHECK (user_one_id < user_two_id),
    UNIQUE(user_one_id, user_two_id)
);

-- 11. MESSAGES TABLE
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- INDEXES
-- ============================================================================
CREATE INDEX idx_incidents_verification ON incidents(verification_status);
CREATE INDEX idx_revisions_incident ON incident_revisions(incident_id, version_number DESC);
CREATE INDEX idx_assets_incident_id ON assets(incident_id);
CREATE INDEX idx_comments_incident_id ON comments(incident_id);
CREATE INDEX idx_incident_culprits_person_id ON incident_culprits(person_id);
CREATE INDEX idx_conversations_users ON conversations(user_one_id, user_two_id);
CREATE INDEX idx_messages_conversation ON messages(conversation_id, sent_at);
CREATE INDEX idx_assets_incident_active ON assets(incident_id) WHERE deleted_at IS NULL;
-- ============================================================================
-- TRIGGERS
-- ============================================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = CURRENT_TIMESTAMP;
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_incidents_updated_at
BEFORE UPDATE ON incidents
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- ROW LEVEL SECURITY (RLS) & SECURITY DEFINER FUNCTIONS
-- ============================================================================

-- Non Super User for RLS testing 
-- Create non-superuser role for application traffic if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'app_user') THEN
        CREATE ROLE app_user WITH LOGIN PASSWORD 'app_password' NOSUPERUSER NOBYPASSRLS;
    END IF;
END
$$;

-- Grant permissions to app_user on public schema and all tables
GRANT ALL PRIVILEGES ON SCHEMA public TO app_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO app_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO app_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO app_user;

-- Helper Function: Check conversation participant securely without recursive RLS issues
CREATE OR REPLACE FUNCTION is_conversation_participant(conv_id UUID, user_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM conversations
        WHERE id = conv_id
          AND (user_one_id = user_id OR user_two_id = user_id)
    );
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
-- Enable and FORCE RLS on Conversations and Messages
ALTER TABLE conversations ENABLE ROW LEVEL SECURITY;
ALTER TABLE conversations FORCE ROW LEVEL SECURITY;

ALTER TABLE messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages FORCE ROW LEVEL SECURITY;

-- 1. Conversation Policy: User must be user_one or user_two
CREATE POLICY conversation_participant_policy ON conversations
    FOR ALL
    USING (
        user_one_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID 
        OR user_two_id = NULLIF(current_setting('app.current_user_id', true), '')::UUID
    );

-- 2. Message Policy: Evaluated cleanly using the helper function
CREATE POLICY message_participant_policy ON messages
    FOR ALL
    USING (
        is_conversation_participant(
            conversation_id, 
            NULLIF(current_setting('app.current_user_id', true), '')::UUID
        )
    );