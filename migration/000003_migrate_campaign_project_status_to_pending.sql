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

SET search_path TO project, public;

-- -----------------------------------------------------
-- 1. Campaign status
-- -----------------------------------------------------

CREATE TYPE project.campaign_status_v2 AS ENUM (
    'PENDING',
    'ACTIVE',
    'PAUSED',
    'ARCHIVED'
);

ALTER TABLE project.campaigns
    ALTER COLUMN status DROP DEFAULT;

ALTER TABLE project.campaigns
    ALTER COLUMN status TYPE project.campaign_status_v2
    USING (
        CASE status::text
            WHEN 'INACTIVE' THEN 'PAUSED'
            WHEN 'PENDING'  THEN 'PENDING'
            WHEN 'ACTIVE'   THEN 'ACTIVE'
            WHEN 'PAUSED'   THEN 'PAUSED'
            WHEN 'ARCHIVED' THEN 'ARCHIVED'
            ELSE 'PENDING'
        END
    )::project.campaign_status_v2;

DROP TYPE project.campaign_status;
ALTER TYPE project.campaign_status_v2 RENAME TO campaign_status;

ALTER TABLE project.campaigns
    ALTER COLUMN status SET DEFAULT 'PENDING'::project.campaign_status;

-- -----------------------------------------------------
-- 2. Project status
-- -----------------------------------------------------

CREATE TYPE project.project_status_v2 AS ENUM (
    'PENDING',
    'ACTIVE',
    'PAUSED',
    'ARCHIVED'
);

ALTER TABLE project.projects
    ALTER COLUMN status DROP DEFAULT;

ALTER TABLE project.projects
    ALTER COLUMN status TYPE project.project_status_v2
    USING (
        CASE status::text
            WHEN 'DRAFT'    THEN 'PENDING'
            WHEN 'PENDING'  THEN 'PENDING'
            WHEN 'ACTIVE'   THEN 'ACTIVE'
            WHEN 'PAUSED'   THEN 'PAUSED'
            WHEN 'ARCHIVED' THEN 'ARCHIVED'
            ELSE 'PENDING'
        END
    )::project.project_status_v2;

DROP TYPE project.project_status;
ALTER TYPE project.project_status_v2 RENAME TO project_status;

ALTER TABLE project.projects
    ALTER COLUMN status SET DEFAULT 'PENDING'::project.project_status;

COMMIT;

-- -----------------------------------------------------
-- Post-checks (run manually if needed)
-- -----------------------------------------------------
-- SELECT unnest(enum_range(NULL::project.campaign_status));
-- SELECT unnest(enum_range(NULL::project.project_status));
-- SELECT status, count(*) FROM project.campaigns GROUP BY 1 ORDER BY 1;
-- SELECT status, count(*) FROM project.projects GROUP BY 1 ORDER BY 1;
