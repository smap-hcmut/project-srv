CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Initial fields
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(255) NOT NULL,
    from_date TIMESTAMP WITH TIME ZONE NOT NULL,
    to_date TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- Additional fields as per your requirements
    brand_name VARCHAR(255) NOT NULL, -- Your brand name
    competitor_names TEXT[], -- Array of competitor names
    brand_keywords TEXT[] NOT NULL, -- Array of your brand's keywords
    competitor_keywords_map JSONB, -- Map/Dictionary storing keywords for each competitor
    -- (Example: '{"Competitor A": ["kwA1", "kwA2"], "Competitor B": ["kwB1"]}')
    
    -- Relational and time management fields
    created_by UUID NOT NULL, -- User ID from JWT token (no FK - different database)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Add index for better query performance
CREATE INDEX IF NOT EXISTS idx_projects_created_by ON projects(created_by);
CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
CREATE INDEX IF NOT EXISTS idx_projects_deleted_at ON projects(deleted_at);