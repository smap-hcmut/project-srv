-- =====================================================
-- Migration: normalize campaign/project lifecycle enums
-- Purpose:
-- - campaign_status  -> PENDING, ACTIVE, PAUSED, ARCHIVED
-- - project_status   -> PENDING, ACTIVE, PAUSED, ARCHIVED
-- - migrate legacy values:
--   * campaign INACTIVE -> PAUSED
--   * project DRAFT     -> PENDING
-- - set new defaults to PENDING
-- =====================================================

BEGIN;

SET search_path TO schema_project, public;

-- -----------------------------------------------------
-- 1. Campaign status
-- -----------------------------------------------------

CREATE TYPE schema_project.campaign_status_v2 AS ENUM (
    'PENDING',
    'ACTIVE',
    'PAUSED',
    'ARCHIVED'
);

ALTER TABLE schema_project.campaigns
    ALTER COLUMN status DROP DEFAULT;

ALTER TABLE schema_project.campaigns
    ALTER COLUMN status TYPE schema_project.campaign_status_v2
    USING (
        CASE status::text
            WHEN 'INACTIVE' THEN 'PAUSED'
            WHEN 'PENDING'  THEN 'PENDING'
            WHEN 'ACTIVE'   THEN 'ACTIVE'
            WHEN 'PAUSED'   THEN 'PAUSED'
            WHEN 'ARCHIVED' THEN 'ARCHIVED'
            ELSE 'PENDING'
        END
    )::schema_project.campaign_status_v2;

DROP TYPE schema_project.campaign_status;
ALTER TYPE schema_project.campaign_status_v2 RENAME TO campaign_status;

ALTER TABLE schema_project.campaigns
    ALTER COLUMN status SET DEFAULT 'PENDING'::schema_project.campaign_status;

-- -----------------------------------------------------
-- 2. Project status
-- -----------------------------------------------------

CREATE TYPE schema_project.project_status_v2 AS ENUM (
    'PENDING',
    'ACTIVE',
    'PAUSED',
    'ARCHIVED'
);

ALTER TABLE schema_project.projects
    ALTER COLUMN status DROP DEFAULT;

ALTER TABLE schema_project.projects
    ALTER COLUMN status TYPE schema_project.project_status_v2
    USING (
        CASE status::text
            WHEN 'DRAFT'    THEN 'PENDING'
            WHEN 'PENDING'  THEN 'PENDING'
            WHEN 'ACTIVE'   THEN 'ACTIVE'
            WHEN 'PAUSED'   THEN 'PAUSED'
            WHEN 'ARCHIVED' THEN 'ARCHIVED'
            ELSE 'PENDING'
        END
    )::schema_project.project_status_v2;

DROP TYPE schema_project.project_status;
ALTER TYPE schema_project.project_status_v2 RENAME TO project_status;

ALTER TABLE schema_project.projects
    ALTER COLUMN status SET DEFAULT 'PENDING'::schema_project.project_status;

COMMIT;

-- -----------------------------------------------------
-- Post-checks (run manually if needed)
-- -----------------------------------------------------
-- SELECT unnest(enum_range(NULL::schema_project.campaign_status));
-- SELECT unnest(enum_range(NULL::schema_project.project_status));
-- SELECT status, count(*) FROM schema_project.campaigns GROUP BY 1 ORDER BY 1;
-- SELECT status, count(*) FROM schema_project.projects GROUP BY 1 ORDER BY 1;
