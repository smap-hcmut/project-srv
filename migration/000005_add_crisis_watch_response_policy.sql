ALTER TYPE project.crisis_status ADD VALUE IF NOT EXISTS 'WATCH';

ALTER TABLE project.projects_crisis_config
    ADD COLUMN IF NOT EXISTS response_policy JSONB DEFAULT '{
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
    }'::jsonb;
