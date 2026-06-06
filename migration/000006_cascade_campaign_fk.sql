-- Convert the campaign_id foreign key on project.projects to ON DELETE
-- CASCADE so deleting a campaign cleans up its projects in one shot instead
-- of leaving orphan rows when the app-level cleanup transaction races with
-- a concurrent campaign delete.
--
-- Constraint name follows the Postgres default for inline-declared FKs.

ALTER TABLE project.projects
    DROP CONSTRAINT IF EXISTS projects_campaign_id_fkey;

ALTER TABLE project.projects
    ADD CONSTRAINT projects_campaign_id_fkey
    FOREIGN KEY (campaign_id) REFERENCES project.campaigns(id) ON DELETE CASCADE;
