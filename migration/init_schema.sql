-- =====================================================
-- Init Schema: schema_project
-- Purpose: Initialize database schema for Project Service (Entity Hierarchy)
-- Matches Pattern: knowledge-srv/migrations
-- Optimized: Uses ENUMs, Indices, and Detailed Comments
-- =====================================================

-- Ensure schema exists
CREATE SCHEMA IF NOT EXISTS schema_project;

-- Grant ownership (optional, good practice)
ALTER SCHEMA schema_project OWNER TO project_master;

-- Set search path to ensure extensions in public are found if needed,
-- though we uses explicit prefixes.
SET search_path TO schema_project, public;

-- Enable pgcrypto for gen_random_uuid() if on older Postgres,
-- otherwise gen_random_uuid() is built-in (PG 13+).
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =====================================================
-- 0. ENUM TYPES
-- =====================================================
-- WARN: DROP TYPE CASCADE will drop columns using these types!
-- This ensures we have clean Enum definitions.

DROP TYPE IF EXISTS schema_project.campaign_status CASCADE;
CREATE TYPE schema_project.campaign_status AS ENUM ('PENDING', 'ACTIVE', 'PAUSED', 'ARCHIVED');

DROP TYPE IF EXISTS schema_project.project_status CASCADE;
CREATE TYPE schema_project.project_status AS ENUM ('PENDING', 'ACTIVE', 'PAUSED', 'ARCHIVED');

DROP TYPE IF EXISTS schema_project.project_config_status CASCADE;
CREATE TYPE schema_project.project_config_status AS ENUM (
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

DROP TYPE IF EXISTS schema_project.entity_type CASCADE;
CREATE TYPE schema_project.entity_type AS ENUM (
    'product', 
    'campaign', 
    'service', 
    'competitor', 
    'topic'
);

-- =====================================================
-- 1. DOMAIN TYPES REGISTRY
-- =====================================================
CREATE TABLE IF NOT EXISTS schema_project.domain_types (
    code VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO schema_project.domain_types (code, name, description)
VALUES
    ('generic', 'Generic', 'Fallback domain used for legacy or uncategorized projects'),
    ('ev', 'Electric Vehicles', 'Electric vehicle products, brands, charging, policy, and ecosystem'),
    ('fnb', 'Food & Beverage', 'Food, beverage, restaurant, cafe, and hospitality conversations'),
    ('crypto', 'Crypto & Blockchain', 'Crypto, blockchain, DeFi, wallets, exchanges, and web3 topics')
ON CONFLICT (code) DO NOTHING;

-- =====================================================
-- 2. CAMPAIGNS TABLE
-- =====================================================
CREATE TABLE IF NOT EXISTS schema_project.campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Enum Status
    status schema_project.campaign_status NOT NULL DEFAULT 'PENDING'::schema_project.campaign_status,
    
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ,
    
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Indices for Campaign
CREATE INDEX IF NOT EXISTS idx_campaigns_status ON schema_project.campaigns(status);
CREATE INDEX IF NOT EXISTS idx_campaigns_created_by ON schema_project.campaigns(created_by);
CREATE INDEX IF NOT EXISTS idx_campaigns_date_range ON schema_project.campaigns(start_date, end_date);

-- =====================================================
-- 3. PROJECTS TABLE
-- =====================================================
CREATE TABLE IF NOT EXISTS schema_project.projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES schema_project.campaigns(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Entity Hierarchy Fields
    brand VARCHAR(100), -- Text field for grouping/filtering in UI
    entity_type schema_project.entity_type NOT NULL DEFAULT 'product'::schema_project.entity_type,
    entity_name VARCHAR(200) NOT NULL, -- Specific name (e.g. "VF8")
    domain_type_code VARCHAR(50) NOT NULL DEFAULT 'generic' REFERENCES schema_project.domain_types(code),
    
    -- Status Fields
    status schema_project.project_status NOT NULL DEFAULT 'PENDING'::schema_project.project_status,
    config_status schema_project.project_config_status DEFAULT 'DRAFT'::schema_project.project_config_status,
    
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Indices for Projects
CREATE INDEX IF NOT EXISTS idx_projects_campaign_id ON schema_project.projects(campaign_id);
CREATE INDEX IF NOT EXISTS idx_projects_status ON schema_project.projects(status);
CREATE INDEX IF NOT EXISTS idx_projects_config_status ON schema_project.projects(config_status);
CREATE INDEX IF NOT EXISTS idx_projects_created_by ON schema_project.projects(created_by);
CREATE INDEX IF NOT EXISTS idx_projects_brand ON schema_project.projects(brand);
CREATE INDEX IF NOT EXISTS idx_projects_entity ON schema_project.projects(entity_type, entity_name);
CREATE INDEX IF NOT EXISTS idx_projects_domain_type_code ON schema_project.projects(domain_type_code);

-- =====================================================
-- Triggers
-- =====================================================
CREATE OR REPLACE FUNCTION schema_project.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_campaigns_updated_at
    BEFORE UPDATE ON schema_project.campaigns
    FOR EACH ROW
    EXECUTE PROCEDURE schema_project.update_updated_at_column();

CREATE TRIGGER update_projects_updated_at
    BEFORE UPDATE ON schema_project.projects
    FOR EACH ROW
    EXECUTE PROCEDURE schema_project.update_updated_at_column();
