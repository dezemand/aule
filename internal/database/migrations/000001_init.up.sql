CREATE SCHEMA IF NOT EXISTS aule;

-- Ensure gen_random_uuid() exists (commonly in public)
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Users table
CREATE TABLE aule.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT,
    name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- User OAuth identities (supports multiple providers per user)
CREATE TABLE aule.user_identities (
    provider VARCHAR(20) NOT NULL,
    sub TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES aule.users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (provider, sub)
);

CREATE INDEX idx_user_identities_user_id ON aule.user_identities(user_id);

-- Refresh tokens
CREATE TABLE aule.refresh_tokens (
    token VARCHAR(80) PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES aule.users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_refresh_tokens_user_id ON aule.refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON aule.refresh_tokens(expires_at);

-- Projects
CREATE TABLE aule.projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    goal TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Project members
CREATE TABLE aule.project_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES aule.projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES aule.users(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, user_id)
);

CREATE INDEX idx_project_members_user_id ON aule.project_members(user_id);
CREATE INDEX idx_project_members_project_id ON aule.project_members(project_id);

-- Updated at trigger function
CREATE OR REPLACE FUNCTION aule.update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at triggers
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON aule.users
    FOR EACH ROW
    EXECUTE FUNCTION aule.update_updated_at();

CREATE TRIGGER update_projects_updated_at
    BEFORE UPDATE ON aule.projects
    FOR EACH ROW
    EXECUTE FUNCTION aule.update_updated_at();
