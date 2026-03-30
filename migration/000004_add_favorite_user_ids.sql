BEGIN;

SET search_path TO schema_project, public;

ALTER TABLE schema_project.campaigns
    ADD COLUMN IF NOT EXISTS favorite_user_ids UUID[] NOT NULL DEFAULT '{}'::uuid[];

ALTER TABLE schema_project.projects
    ADD COLUMN IF NOT EXISTS favorite_user_ids UUID[] NOT NULL DEFAULT '{}'::uuid[];

CREATE INDEX IF NOT EXISTS idx_campaigns_favorite_user_ids_gin
    ON schema_project.campaigns
    USING GIN (favorite_user_ids);

CREATE INDEX IF NOT EXISTS idx_projects_favorite_user_ids_gin
    ON schema_project.projects
    USING GIN (favorite_user_ids);

COMMIT;
