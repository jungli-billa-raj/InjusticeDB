-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ENUM Types
CREATE TYPE user_role AS ENUM ('user', 'moderator', 'admin');
CREATE TYPE entity_type AS ENUM ('individual', 'group', 'organization');
CREATE TYPE verification_status_type AS ENUM ('pending', 'verified', 'rejected', 'disputed');
CREATE TYPE justice_status_type AS ENUM ('proceeding', 'served', 'stalled');
CREATE TYPE culprit_status_type AS ENUM ('suspect', 'accused', 'guilty', 'convicted');
CREATE TYPE asset_type AS ENUM ('image', 'video', 'article', 'archive_link');

-- 1. USERS TABLE (With Ranks / Credibility)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    auth_provider VARCHAR(50) NOT NULL DEFAULT 'email',
    role user_role NOT NULL DEFAULT 'user',
    credibility_score INTEGER NOT NULL DEFAULT 100 CHECK (credibility_score >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 2. PEOPLE / ORGANIZATIONS TABLE
CREATE TABLE people (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(150) NOT NULL,
    organization VARCHAR(150),
    age INTEGER CHECK (age >= 0),
    state VARCHAR(100),
    city VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 3. INCIDENTS TABLE (Master Record)
CREATE TABLE incidents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    full_story TEXT NOT NULL,
    severity INTEGER NOT NULL CHECK (severity >= 1 AND severity <= 10),
    state VARCHAR(100) NOT NULL,
    city VARCHAR(100) NOT NULL,
    verification_status verification_status_type NOT NULL DEFAULT 'pending',
    justice_status justice_status_type NOT NULL DEFAULT 'proceeding',
    current_version INTEGER NOT NULL DEFAULT 1,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 4. INCIDENT REVISIONS TABLE (Git-style Version History)
CREATE TABLE incident_revisions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_id UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    title VARCHAR(255) NOT NULL,
    full_story TEXT NOT NULL,
    change_summary TEXT NOT NULL, -- e.g., "Updated full story with eyewitness account"
    edited_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(incident_id, version_number)
);

-- 5. JUNCTION TABLE (Incidents <-> Culprits)
CREATE TABLE incident_culprits (
    incident_id UUID REFERENCES incidents(id) ON DELETE CASCADE,
    person_id UUID REFERENCES people(id) ON DELETE CASCADE,
    culprit_status culprit_status_type NOT NULL DEFAULT 'suspect',
    PRIMARY KEY (incident_id, person_id)
);

-- 6. ASSETS & SOURCES TABLE
CREATE TABLE assets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_id UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    type asset_type NOT NULL,
    url TEXT NOT NULL,
    archive_url TEXT, -- Saved Wayback / Archive.is link
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 7. CONVERSATIONS TABLE (1-on-1 DMs)
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_one_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_two_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_one_id, user_two_id)
);

-- 8. MESSAGES TABLE
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- INDEXES FOR SPEED
CREATE INDEX idx_revisions_incident ON incident_revisions(incident_id, version_number);
CREATE INDEX idx_messages_conversation ON messages(conversation_id, sent_at);
CREATE INDEX idx_incidents_location ON incidents(state, city);