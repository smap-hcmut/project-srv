# Project/Campaign Favorite Proposal

## 1. Context

Current `project-srv` manages two top-level resources:

- `campaigns`
- `projects`

Both resources already have authenticated CRUD APIs and can read `user_id` from auth context. However, there is no per-user personalization layer yet.

The new requirement is:

- allow many users to `pin/unpin` or `favorite/unfavorite` a `campaign`
- allow many users to `pin/unpin` or `favorite/unfavorite` a `project`
- each `campaign` and `project` must keep a list of user IDs that have favorited it

## 2. Recommendation

Recommend using the term `favorite` in backend and data model.

Why:

- `favorite` is the stored relation
- UI can still call the action `Pin` / `Unpin`
- backend only needs to answer:
  - which users favorited this resource
  - is current user in that list

Suggested product semantics:

- backend stores `favorite_user_ids`
- UI may label the action as `Pin` / `Unpin`

## 3. Current State

Relevant current observations from `project-srv`:

- `campaigns` and `projects` tables only store entity data
- authenticated handlers already use `mw.Auth()`
- usecases already read `user_id` from context for `created_by`
- list/detail responses currently do not expose favorite state
- current list ordering is generic and not personalized

## 4. Scope

### In scope

- favorite/unfavorite `campaign`
- favorite/unfavorite `project`
- store favorite users directly on each entity
- expose `is_favorite` in `detail`
- expose `is_favorite` in `list`
- support optional `favorite_only=true`
- support sorting favorites first in list APIs

### Out of scope for phase 1

- arbitrary pin ordering per user
- folders/collections
- cross-service notification/eventing
- global favorite analytics
- full audit history of who favorited/unfavorited when

## 5. Data Model Proposal

Use array-based storage directly on the main tables.

### Campaign

Add column to `schema_project.campaigns`:

- `favorite_user_ids UUID[] NOT NULL DEFAULT '{}'::uuid[]`

Recommended index:

- `CREATE INDEX ... USING GIN (favorite_user_ids)`

### Project

Add column to `schema_project.projects`:

- `favorite_user_ids UUID[] NOT NULL DEFAULT '{}'::uuid[]`

Recommended index:

- `CREATE INDEX ... USING GIN (favorite_user_ids)`

### Why choose array storage

This proposal now assumes:

- the product wants each entity to directly carry the list of users who favorited it
- favorite is lightweight metadata, not a rich standalone relation
- expected favorite cardinality per resource is moderate

Benefits:

- schema simple
- no extra join table
- detail response can be enriched directly from one row
- easier mental model for backend and frontend

Trade-offs:

- row update contention if many users favorite the same item concurrently
- array can grow large if a resource is favorited by many users
- weaker auditability because favorite action history is not modeled as rows
- later analytics like "top favorited by week" will be harder

Recommendation:

- acceptable for phase 1 if favorite list size stays moderate
- if favorite cardinality grows large later, migrate to join tables

## 6. API Proposal

### Campaign APIs

#### Add endpoints

- `POST /campaigns/:id/favorite`
- `DELETE /campaigns/:id/favorite`

Alternative alias if frontend strongly prefers pin wording:

- `POST /campaigns/:id/pin`
- `DELETE /campaigns/:id/pin`

Recommendation:

- expose only one canonical backend naming
- prefer `favorite`

#### Update responses

Add to `campaignResp`:

- `is_favorite bool`
- optional internal-only field in domain model: `favorite_user_ids []string`

#### Update list API

Extend `GET /campaigns` query params:

- `favorite_only=true|false`
- `sort=favorite_desc|created_at_desc`

### Project APIs

#### Add endpoints

- `POST /projects/:project_id/favorite`
- `DELETE /projects/:project_id/favorite`

#### Update responses

Add to `projectResp`:

- `is_favorite bool`
- optional internal-only field in domain model: `favorite_user_ids []string`

#### Update list API

Extend `GET /campaigns/:id/projects` query params:

- `favorite_only=true|false`
- `sort=favorite_desc|created_at_desc`

## 7. Domain and UseCase Changes

### Campaign module

Need to add:

- `Favorite(ctx, campaignID string) error`
- `Unfavorite(ctx, campaignID string) error`

List/detail outputs need user-personalized enrichment:

- `Campaign` should know whether current user belongs to `favorite_user_ids`

Recommendation:

- add `FavoriteUserIDs []string` to `model.Campaign`
- compute `IsFavorite bool` at response mapping or usecase level

### Project module

Need to add:

- `Favorite(ctx, projectID string) error`
- `Unfavorite(ctx, projectID string) error`

Recommendation:

- add `FavoriteUserIDs []string` to `model.Project`
- compute `IsFavorite bool` at response mapping or usecase level

## 8. Repository Changes

### Campaign repository

Add methods:

- `Favorite(ctx, campaignID, userID string) error`
- `Unfavorite(ctx, campaignID, userID string) error`

Favorite update behavior:

- append user ID only if not already present
- remove user ID if present

List/detail query behavior:

- detail reads `favorite_user_ids`
- list can compute favorite state using current user and array membership
- `favorite_only=true` can filter by array containment

Suggested SQL style:

- favorite:
  - `array_append(...)` with duplicate guard
- unfavorite:
  - `array_remove(...)`
- filter:
  - `favorite_user_ids @> ARRAY[?]::uuid[]`

### Project repository

Add methods:

- `Favorite(ctx, projectID, userID string) error`
- `Unfavorite(ctx, projectID, userID string) error`

Use same array-based approach as campaign.

## 9. Handler Changes

### New handlers

Campaign:

- `Favorite`
- `Unfavorite`

Project:

- `Favorite`
- `Unfavorite`

### Presenter changes

Campaign request/query DTO:

- add `favorite_only`
- add optional `sort`

Project request/query DTO:

- add `favorite_only`
- add optional `sort`

Campaign response DTO:

- add `is_favorite`

Project response DTO:

- add `is_favorite`

Recommendation:

- do not expose raw `favorite_user_ids` in public response for now
- keep it internal unless product explicitly needs the full list

## 10. Migration Changes

Need one new migration, for example:

- `000003_add_favorite_user_ids.sql`

Migration contents:

- add `favorite_user_ids UUID[] NOT NULL DEFAULT '{}'::uuid[]` to `campaigns`
- add `favorite_user_ids UUID[] NOT NULL DEFAULT '{}'::uuid[]` to `projects`
- add GIN indices on both columns

No backfill needed.

## 11. Auth and Access Rules

Use `user_id` from auth context as the favorite actor.

Phase 1 rule:

- any authenticated user can favorite/unfavorite any campaign/project they can access through existing API

## 12. Response Shape Example

### Campaign detail

```json
{
  "campaign": {
    "id": "campaign-uuid",
    "name": "Q1 2026 VinFast Campaign",
    "status": "ACTIVE",
    "created_by": "user-uuid",
    "created_at": "2026-03-28T10:00:00Z",
    "updated_at": "2026-03-28T10:00:00Z",
    "is_favorite": true
  }
}
```

### Project list item

```json
{
  "id": "project-uuid",
  "campaign_id": "campaign-uuid",
  "name": "VinFast VF8 Monitoring",
  "entity_type": "product",
  "entity_name": "VF8",
  "status": "ACTIVE",
  "created_by": "user-uuid",
  "created_at": "2026-03-28T10:00:00Z",
  "updated_at": "2026-03-28T10:00:00Z",
  "is_favorite": false
}
```

## 13. Implementation Checklist

### Database

- add `favorite_user_ids` to `campaigns`
- add `favorite_user_ids` to `projects`
- add GIN indices
- regenerate sqlboiler models if needed

### Campaign module

- add favorite methods to `campaign.UseCase`
- add repository methods to update arrays
- add handlers and routes
- add `is_favorite` to response DTO
- add `favorite_only` and optional sort handling in list

### Project module

- add favorite methods to `project.UseCase`
- add repository methods to update arrays
- add handlers and routes
- add `is_favorite` to response DTO
- add `favorite_only` and optional sort handling in list

### Tests

- favorite success
- unfavorite success
- idempotent favorite
- idempotent unfavorite
- 404 when resource not found
- list with `favorite_only=true`
- detail returns correct `is_favorite`
- duplicate user ID is not inserted twice

## 14. Suggested Rollout Plan

### Phase 1

- migration add array columns
- favorite/unfavorite endpoints
- `is_favorite` in detail/list
- `favorite_only` filter

### Phase 2

- favorites-first sorting by default if product wants it
- optional global `/projects?favorite_only=true`

### Phase 3

- if favorite arrays become too large:
  - migrate to normalized join tables
  - keep response contract the same

## 15. Risks

### Row contention

If many users favorite the same campaign/project at the same time, one row may be updated frequently.

### Array growth

If one resource is favorited by many users, row size and update cost will grow.

### Audit limitation

Array model stores current favorite state, not detailed history.

### Query flexibility

Later analytics around favorite actions may be harder than with normalized tables.

## 16. Final Recommendation

Implement favorite as:

- `campaigns.favorite_user_ids UUID[]`
- `projects.favorite_user_ids UUID[]`

Use:

- `POST/DELETE .../favorite` for write operations
- `is_favorite` in list/detail for personalized UX

This design is the simplest version that matches the current requirement that each `campaign` and `project` directly stores the list of user IDs who favorited it.
