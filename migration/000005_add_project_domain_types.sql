BEGIN;

SET search_path TO schema_project, public;

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
ON CONFLICT (code) DO UPDATE
SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    is_active = TRUE,
    updated_at = NOW();

ALTER TABLE schema_project.projects
    ADD COLUMN IF NOT EXISTS domain_type_code VARCHAR(50);

UPDATE schema_project.projects
SET domain_type_code = 'generic'
WHERE domain_type_code IS NULL OR BTRIM(domain_type_code) = '';

ALTER TABLE schema_project.projects
    ALTER COLUMN domain_type_code SET DEFAULT 'generic';

ALTER TABLE schema_project.projects
    ALTER COLUMN domain_type_code SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'projects_domain_type_code_fkey'
          AND connamespace = 'schema_project'::regnamespace
    ) THEN
        ALTER TABLE schema_project.projects
            ADD CONSTRAINT projects_domain_type_code_fkey
            FOREIGN KEY (domain_type_code)
            REFERENCES schema_project.domain_types(code);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_projects_domain_type_code
    ON schema_project.projects (domain_type_code);

COMMIT;
