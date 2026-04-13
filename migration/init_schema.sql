-- =====================================================
-- Init Schema: project
-- Purpose: Initialize database schema for Project Service (Entity Hierarchy)
-- Matches Pattern: knowledge-srv/migrations
-- Optimized: Uses ENUMs, Indices, and Detailed Comments
-- =====================================================

-- Ensure schema exists
CREATE SCHEMA IF NOT EXISTS project;

-- Grant ownership (optional, good practice)
ALTER SCHEMA project OWNER TO project_master;

-- Set search path to ensure extensions in public are found if needed,
-- though we uses explicit prefixes.
SET search_path TO project, public;

-- Enable pgcrypto for gen_random_uuid() if on older Postgres,
-- otherwise gen_random_uuid() is built-in (PG 13+).
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =====================================================
-- 0. ENUM TYPES
-- =====================================================
-- WARN: DROP TYPE CASCADE will drop columns using these types!
-- This ensures we have clean Enum definitions.

DROP TYPE IF EXISTS project.campaign_status CASCADE;
CREATE TYPE project.campaign_status AS ENUM ('PENDING', 'ACTIVE', 'PAUSED', 'ARCHIVED');

DROP TYPE IF EXISTS project.project_status CASCADE;
CREATE TYPE project.project_status AS ENUM ('PENDING', 'ACTIVE', 'PAUSED', 'ARCHIVED');

DROP TYPE IF EXISTS project.project_config_status CASCADE;
CREATE TYPE project.project_config_status AS ENUM (
    'DRAFT', 
    'CONFIGURING', 
    'ONBOARDING', 
    'ONBOARDING_DONE', 
    'DRYRUN_RUNNING', 
    'DRYRUN_SUCCESS', 
    'DRYRUN_FAILED', 
    'ACTIVE', 
    'ERROR'
);

DROP TYPE IF EXISTS project.entity_type CASCADE;
CREATE TYPE project.entity_type AS ENUM (
    'product', 
    'campaign', 
    'service', 
    'competitor', 
    'topic'
);

-- =====================================================
-- 1. CAMPAIGNS TABLE
-- =====================================================
CREATE TABLE IF NOT EXISTS project.campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Enum Status
    status project.campaign_status NOT NULL DEFAULT 'PENDING'::project.campaign_status,
    
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ,
    
    created_by UUID NOT NULL,
    favorite_user_ids UUID[] NOT NULL DEFAULT '{}'::uuid[],
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Indices for Campaign
CREATE INDEX IF NOT EXISTS idx_campaigns_status ON project.campaigns(status);
CREATE INDEX IF NOT EXISTS idx_campaigns_created_by ON project.campaigns(created_by);
CREATE INDEX IF NOT EXISTS idx_campaigns_date_range ON project.campaigns(start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_campaigns_favorite_user_ids_gin
    ON project.campaigns USING GIN (favorite_user_ids);

-- =====================================================
-- 3. PROJECTS TABLE
-- =====================================================
CREATE TABLE IF NOT EXISTS project.projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES project.campaigns(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Entity Hierarchy Fields
    brand VARCHAR(100), -- Text field for grouping/filtering in UI
    entity_type project.entity_type NOT NULL DEFAULT 'product'::project.entity_type,
    entity_name VARCHAR(200) NOT NULL, -- Specific name (e.g. "VF8")
    domain_type_code VARCHAR(50) NOT NULL DEFAULT '_default',
    
    -- Status Fields
    status project.project_status NOT NULL DEFAULT 'PENDING'::project.project_status,
    config_status project.project_config_status DEFAULT 'DRAFT'::project.project_config_status,
    
    created_by UUID NOT NULL,
    favorite_user_ids UUID[] NOT NULL DEFAULT '{}'::uuid[],
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Indices for Projects
CREATE INDEX IF NOT EXISTS idx_projects_campaign_id ON project.projects(campaign_id);
CREATE INDEX IF NOT EXISTS idx_projects_status ON project.projects(status);
CREATE INDEX IF NOT EXISTS idx_projects_config_status ON project.projects(config_status);
CREATE INDEX IF NOT EXISTS idx_projects_created_by ON project.projects(created_by);
CREATE INDEX IF NOT EXISTS idx_projects_brand ON project.projects(brand);
CREATE INDEX IF NOT EXISTS idx_projects_entity ON project.projects(entity_type, entity_name);
CREATE INDEX IF NOT EXISTS idx_projects_domain_type_code ON project.projects(domain_type_code);
CREATE INDEX IF NOT EXISTS idx_projects_favorite_user_ids_gin
    ON project.projects USING GIN (favorite_user_ids);

-- =====================================================
-- Triggers
-- =====================================================
CREATE OR REPLACE FUNCTION project.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_campaigns_updated_at
    BEFORE UPDATE ON project.campaigns
    FOR EACH ROW
    EXECUTE PROCEDURE project.update_updated_at_column();

CREATE TRIGGER update_projects_updated_at
    BEFORE UPDATE ON project.projects
    FOR EACH ROW
    EXECUTE PROCEDURE project.update_updated_at_column();
