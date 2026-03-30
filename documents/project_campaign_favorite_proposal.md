# Project/Campaign Favorite Proposal

## 1. Context

Current `project-srv` manages two top-level resources:

- `campaigns`
- `projects`

Both resources already have authenticated CRUD APIs and can read `user_id` from request context via auth middleware. However, there is no per-user personalization layer yet.

The new requirement is:

- allow a user to mark a `project` or `campaign` as `pin/unpin`
- or equivalently `favorite/unfavorite`
- so the UI can quickly surface items important to that specific user

## 2. Recommendation

Recommend using the term `favorite` in backend and data model.

Why:

- `favorite` is a simple boolean relation between `user` and `resource`
- `pin` usually implies visual placement or ordering behavior in UI
- backend can still support UI pinning by sorting favorites first
- later, if true pin-order is needed, we can extend with `position` without renaming the concept

Suggested product semantics:

- backend stores `favorite`
- UI may label the action as `Pin` / `Unpin` if desired

## 3. Current State

Relevant current observations from `project-srv`:

- `campaigns` and `projects` tables only store entity data, not per-user preferences
- authenticated handlers already use `mw.Auth()`
- usecases already read `user_id` from context for `created_by`
- list/detail responses currently do not expose any user-personalized fields
- current list ordering is generic and not personalized

## 4. Scope

### In scope

- favorite/unfavorite `campaign`
- favorite/unfavorite `project`
- expose `is_favorite` in `detail`
- expose `is_favorite` in `list`
- support optional `favorite_only=true`
- support sorting favorites first in list APIs

### Out of scope for phase 1

- arbitrary pin ordering per user
- folders/collections
- cross-service notification/eventing
- favorite counts across users
- sharing favorite sets between users

## 5. Data Model Proposal

Do not add `is_favorite` directly into `campaigns` or `projects`.

Reason:

- favorite is per-user state
- one campaign/project can be favorited by many users
- storing it on the main table would be incorrect

### Option A: Recommended

Create two explicit join tables:

#### `schema_project.campaign_favorites`

- `campaign_id UUID NOT NULL REFERENCES schema_project.campaigns(id) ON DELETE CASCADE`
- `user_id UUID NOT NULL`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- primary key: `(campaign_id, user_id)`
- index: `(user_id, created_at DESC)`

#### `schema_project.project_favorites`

- `project_id UUID NOT NULL REFERENCES schema_project.projects(id) ON DELETE CASCADE`
- `user_id UUID NOT NULL`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- primary key: `(project_id, user_id)`
- index: `(user_id, created_at DESC)`

Why this option is best here:

- simple SQL
- clear FK integrity
- fits existing service style
- no polymorphic FK complexity

### Option B: Generic table

One generic table like `user_entity_favorites(entity_kind, entity_id, user_id, ...)`.

Not recommended for now because:

- weak FK integrity
- more conditional query logic
- harder to keep clean with sqlboiler and repository code

## 6. API Proposal

### Campaign APIs

#### Add endpoints

- `POST /campaigns/:id/favorite`
- `DELETE /campaigns/:id/favorite`

Alternative alias if frontend strongly prefers pin wording:

- `POST /campaigns/:id/pin`
- `DELETE /campaigns/:id/pin`

Recommendation:

- expose only one canonical API name in backend
- prefer `favorite`

#### Update responses

Add to `campaignResp`:

- `is_favorite bool`
- optional: `favorited_at *string`

#### Update list API

Extend `GET /campaigns` query params:

- `favorite_only=true|false`
- `sort=favorite_desc|created_at_desc`

Default behavior:

- preserve current behavior unless explicitly requested

Optional better UX behavior:

- authenticated list sorts favorites first, then `created_at DESC`

This is good for product UX, but changes existing behavior. If we want low risk rollout, keep it opt-in first.

### Project APIs

#### Add endpoints

- `POST /projects/:project_id/favorite`
- `DELETE /projects/:project_id/favorite`

#### Update responses

Add to `projectResp`:

- `is_favorite bool`
- optional: `favorited_at *string`

#### Update list API

Extend `GET /campaigns/:id/projects` query params:

- `favorite_only=true|false`
- `sort=favorite_desc|created_at_desc`

Optional future:

- `GET /projects?favorite_only=true`

Current service does not expose global project list outside campaign nesting, so this is phase 2 unless needed now.

## 7. Domain and UseCase Changes

### Campaign module

Need to add:

- `Favorite(ctx, campaignID string) error`
- `Unfavorite(ctx, campaignID string) error`

Likely new inputs:

- `FavoriteInput { CampaignID string }`
- `UnfavoriteInput { CampaignID string }`

List/detail outputs need user-personalized enrichment:

- `Campaign` should include favorite metadata, or
- response mapper should enrich using a separate favorite lookup result

Recommendation:

- add `IsFavorite bool` and `FavoritedAt *time.Time` to `model.Campaign`

### Project module

Need to add:

- `Favorite(ctx, projectID string) error`
- `Unfavorite(ctx, projectID string) error`

Need response enrichment for list/detail:

- add `IsFavorite bool`
- add optional `FavoritedAt *time.Time`

Recommendation:

- add `IsFavorite bool` and `FavoritedAt *time.Time` to `model.Project`

## 8. Repository Changes

### Campaign repository

Add methods:

- `Favorite(ctx, campaignID, userID string) error`
- `Unfavorite(ctx, campaignID, userID string) error`
- `GetFavoriteMap(ctx, userID string, campaignIDs []string) (map[string]FavoriteMeta, error)`

Update list/detail queries:

- for detail: left join favorites table by `(campaign_id, user_id)`
- for list: either left join or batch lookup favorite map after fetching campaigns

Recommendation:

- use batch lookup instead of embedding complex joins into every list query initially
- keeps current repo query shape simpler

### Project repository

Add methods:

- `Favorite(ctx, projectID, userID string) error`
- `Unfavorite(ctx, projectID, userID string) error`
- `GetFavoriteMap(ctx, userID string, projectIDs []string) (map[string]FavoriteMeta, error)`

Update list/detail flow similarly.

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
- add optional `favorited_at`

Project response DTO:

- add `is_favorite`
- add optional `favorited_at`

## 10. Migration Changes

Need one new migration, for example:

- `000003_add_favorites.sql`

Migration contents:

- create `campaign_favorites`
- create `project_favorites`
- primary keys / indices

No backfill needed.

## 11. Auth and Access Rules

Use `user_id` from auth context as the owner of favorite state.

Phase 1 rule:

- any authenticated user can favorite/unfavorite any campaign/project they can access through existing API

If resource-level authorization is introduced later, favorite APIs should reuse the same permission check.

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
    "is_favorite": true,
    "favorited_at": "2026-03-28T11:00:00Z"
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

- add migration for `campaign_favorites`
- add migration for `project_favorites`
- regenerate sqlboiler models if project uses generated DB bindings from schema

### Campaign module

- add favorite methods to `campaign.UseCase`
- add favorite repository methods
- add handlers and routes
- add `is_favorite` to response DTO
- add `favorite_only` and optional sort handling in list

### Project module

- add favorite methods to `project.UseCase`
- add favorite repository methods
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
- list returns correct `is_favorite` for mixed resources

## 14. Suggested Rollout Plan

### Phase 1

- DB tables
- favorite/unfavorite endpoints
- `is_favorite` in detail/list
- `favorite_only` filter

### Phase 2

- favorites-first sorting by default
- optional global `/projects` favorite list
- optional `favorited_at` sort

### Phase 3

- true pin ordering with `position`
- drag/drop ordering in UI

## 15. Risks

### Behavior drift in list APIs

If we sort favorites first by default, existing clients may see changed ordering.

Mitigation:

- start with opt-in sort param

### N+1 favorite lookups

If we enrich favorite state per item one-by-one, list performance will degrade.

Mitigation:

- batch lookup favorite map by resource IDs

### Naming confusion: pin vs favorite

Frontend/product may say pin, backend may say favorite.

Mitigation:

- align terminology early
- document that pin is a UI behavior backed by favorite state

## 16. Final Recommendation

Implement `favorite/unfavorite` first, not true `pin`.

Concrete recommendation:

- add `campaign_favorites`
- add `project_favorites`
- expose `POST/DELETE .../favorite`
- include `is_favorite` in list/detail
- optionally add `favorite_only=true`

This gives the product team a working pin-like experience quickly, with low schema risk and clean extensibility later.
