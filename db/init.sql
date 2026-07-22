-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Define Custom Native ENUM Types
CREATE TYPE user_role AS ENUM ('user', 'moderator', 'admin');
CREATE TYPE verification_status_type AS ENUM ('pending', 'verified', 'rejected');
CREATE TYPE justice_status_type AS ENUM ('proceeding', 'served', 'stalled');
CREATE TYPE culprit_status_type AS ENUM ('suspect', 'accused', 'guilty', 'convicted');
CREATE TYPE asset_type AS ENUM ('image', 'video', 'article', 'archive_link');

-- USERS TABLE
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    auth_provider VARCHAR(50) NOT NULL DEFAULT 'email',
    role user_role NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- PEOPLE / ORGANIZATIONS TABLE
CREATE TABLE people (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(150) NOT NULL,
    organization VARCHAR(150),
    age INTEGER CHECK (age >= 0),
    state VARCHAR(100),
    city VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- INCIDENTS TABLE
CREATE TABLE incidents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    full_story TEXT NOT NULL,
    severity INTEGER NOT NULL CHECK (severity >= 1 AND severity <= 10),
    state VARCHAR(100) NOT NULL,
    city VARCHAR(100) NOT NULL,
    verification_status verification_status_type NOT NULL DEFAULT 'pending',
    justice_status justice_status_type NOT NULL DEFAULT 'proceeding',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- JUNCTION TABLE (Many-to-Many: Incidents <-> People/Entities)
CREATE TABLE incident_culprits (
    incident_id UUID REFERENCES incidents(id) ON DELETE CASCADE,
    person_id UUID REFERENCES people(id) ON DELETE CASCADE, 
    culprit_status culprit_status_type NOT NULL DEFAULT 'suspect',
    PRIMARY KEY (incident_id, person_id)
);

-- ASSETS TABLE
CREATE TABLE assets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_id UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    type asset_type NOT NULL,
    url TEXT NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- COMMENTS TABLE
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_id UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- YDCIDC TARGETS TABLE
CREATE TABLE ydcidc_targets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(150) NOT NULL,
    occupation VARCHAR(100),
    state VARCHAR(100),
    city VARCHAR(100),
    cause_of_resentment TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);


-- INDEXES FOR FAST QUERYING & FILTERING
CREATE INDEX idx_incidents_location ON incidents(state, city);
CREATE INDEX idx_incidents_verification ON incidents(verification_status);
CREATE INDEX idx_assets_incident_id ON assets(incident_id);
CREATE INDEX idx_comments_incident_id ON comments(incident_id);
CREATE INDEX idx_incident_culprits_person_id ON incident_culprits(person_id);

-- AUTOMATIC UPDATED_AT TRIGGER FUNCTION
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