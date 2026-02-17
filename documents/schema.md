# Project Service Database Schema (v2.1)

**Schema Name:** `schema_project`

This schema adheres to the **SMAP V2 Architecture** (Entity Hierarchy) and uses **PostgreSQL ENUMs** for strict type safety.

## ENUM Types

### 1. `campaign_status`

- `ACTIVE`: Running normally.
- `INACTIVE`: Stopped/Paused.
- `ARCHIVED`: Soft-deleted or hidden from main view.

### 2. `project_status`

- `ACTIVE`: Operational, monitoring is running.
- `PAUSED`: Temporarily stopped monitoring.
- `ARCHIVED`: Soft-deleted.

### 3. `project_config_status`

Tracks the Multi-step Wizard progress:

- `DRAFT`: Initial state.
- `CONFIGURING`: Setting up data sources.
- `ONBOARDING`: Passive sources (Files/Webhook) require AI mapping.
- `ONBOARDING_DONE`: Mapping confirmed by user.
- `DRYRUN_RUNNING`: Crawling test data (for Crawl sources).
- `DRYRUN_SUCCESS`: Dry run passed, ready to activate.
- `DRYRUN_FAILED`: Dry run encountered errors.
- `ACTIVE`: Setup complete, project is live.
- `ERROR`: Critical setup error.

### 4. `entity_type`

- `product`: Monitor a specific product (e.g., "VF8").
- `campaign`: Monitor a marketing campaign.
- `service`: Monitor a service quality.
- `competitor`: Monitor a competitor brand/product.
- `topic`: Monitor a general topic.

### 5. `crisis_status`

- `NORMAL`: No issues detected.
- `WARNING`: Early warning signs (e.g., volume spike).
- `CRITICAL`: Confirmed crisis (e.g., legal keywords, extreme sentiment drop).

## Tables

### 1. `campaigns`

Top-level grouping entity. Represents a logical analysis unit.

| Column        | Type                       | Description                              |
| :------------ | :------------------------- | :--------------------------------------- |
| `id`          | UUID                       | Primary Key (Default: gen_random_uuid()) |
| `name`        | VARCHAR(255)               | Name of the campaign                     |
| `description` | TEXT                       | Optional description                     |
| `status`      | **ENUM** `campaign_status` | Default: `ACTIVE`                        |
| `start_date`  | TIMESTAMP WITH TIME ZONE   | Campaign start                           |
| `end_date`    | TIMESTAMP WITH TIME ZONE   | Campaign end                             |
| `created_by`  | UUID                       | User ID (Owner)                          |
| `created_at`  | TIMESTAMP WITH TIME ZONE   | Creation time (Default: NOW())           |
| `updated_at`  | TIMESTAMP WITH TIME ZONE   | Last update time                         |
| `deleted_at`  | TIMESTAMP WITH TIME ZONE   | Soft delete timestamp                    |

### 2. `projects`

Child entity of a Campaign. Represents a specific **Entity Monitoring Unit**.

| Column              | Type                             | Description                               |
| :------------------ | :------------------------------- | :---------------------------------------- |
| `id`                | UUID                             | Primary Key (Default: gen_random_uuid())  |
| `campaign_id`       | UUID                             | Foreign Key to `campaigns.id`             |
| `name`              | VARCHAR(255)                     | Project Name                              |
| `description`       | TEXT                             | Description                               |
| `status`            | **ENUM** `project_status`        | Default: `ACTIVE`                         |
| **`brand`**         | VARCHAR(100)                     | Brand name for grouping (e.g., "VinFast") |
| **`entity_type`**   | **ENUM** `entity_type`           | Categorization of the monitored subject   |
| **`entity_name`**   | VARCHAR(200)                     | Specific entity name (e.g., "VF8")        |
| **`config_status`** | **ENUM** `project_config_status` | Wizard Status (Default: `DRAFT`)          |
| `created_by`        | UUID                             | User ID (Owner)                           |
| `created_at`        | TIMESTAMP WITH TIME ZONE         | Creation time (Default: NOW())            |
| `updated_at`        | TIMESTAMP WITH TIME ZONE         | Last update time                          |
| `deleted_at`        | TIMESTAMP WITH TIME ZONE         | Soft delete timestamp                     |

## Relationships

- `campaigns` (1) -> (N) `projects`

### 3. `project_crisis_config`

Stores 1-to-1 crisis detection rules for a project.

| Column             | Type                     | Description                                       |
| :----------------- | :----------------------- | :------------------------------------------------ |
| `project_id`       | UUID                     | Primary Key, FK to `projects.id`                  |
| `status`           | **ENUM** `crisis_status` | `NORMAL`, `WARNING`, `CRITICAL`                   |
| `keywords_rules`   | JSONB                    | Critical, Legal/Health, Slang terms               |
| `volume_rules`     | JSONB                    | Thresholds for volume spikes (150% = CRITICAL)    |
| `sentiment_rules`  | JSONB                    | Thresholds for negative sentiment (15% = HYGIENE) |
| `influencer_rules` | JSONB                    | Thresholds for KOL reach/virality                 |
| `updated_at`       | TIMESTAMP WITH TIME ZONE | Last update time                                  |
