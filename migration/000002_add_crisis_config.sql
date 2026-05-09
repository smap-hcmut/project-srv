-- Create ENUM for Crisis Status
CREATE TYPE project.crisis_status AS ENUM ('NORMAL', 'WATCH', 'WARNING', 'CRITICAL');

-- project_crisis_config table
-- Stores crisis detection rules for each project
CREATE TABLE IF NOT EXISTS project.projects_crisis_config (
    project_id UUID PRIMARY KEY REFERENCES project.projects(id) ON DELETE CASCADE,
    
    -- Crisis Status: NORMAL, WATCH, WARNING, CRITICAL
    status project.crisis_status DEFAULT 'NORMAL',
    
    -- Keyword Rules (JSONB)
    -- { "critical": ["scam", "fake"], "legal": ["police", "court"], "slang": ["phot", "boc phot"] }
    keywords_rules JSONB,

    -- Volume Rules (JSONB)
    -- { "warning_threshold": 1.5, "critical_threshold": 2.5, "baseline_days": 7 }
    volume_rules JSONB,

    -- Sentiment Rules (JSONB)
    -- { "min_sample_size": 50, "aspect_thresholds": { "hygiene": 0.15, "safety": 0.15 } }
    sentiment_rules JSONB,

    -- Influencer Rules (JSONB)
    -- { "min_followers": 50000, "viral_share_count": 1000 }
    influencer_rules JSONB,

    -- Response policy (JSONB)
    -- Controls adaptive crawling and user notification thresholds independently.
    response_policy JSONB DEFAULT '{
      "adaptive_crawl": {
        "enabled": true,
        "trigger_level": "WATCH",
        "cooldown_minutes": 30
      },
      "notification": {
        "enabled": true,
        "trigger_level": "WARNING",
        "repeat_cooldown_minutes": 60,
        "ops_alert_on_critical": true
      }
    }'::jsonb,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for fast lookup by status (to find projects in crisis)
CREATE INDEX idx_projects_crisis_config_status ON project.projects_crisis_config(status);
