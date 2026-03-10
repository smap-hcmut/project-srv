# SMAP Project Service

## Document Status

- **Status:** Derived
- **Canonical reference:** `/mnt/f/SMAP_v2/cross-service-docs/proposal_chuan_hoa_docs_3_service_v1.md`
- **Last updated:** 05/03/2026

## Overview

`project-srv` là service quản lý nghiệp vụ:
- Campaign lifecycle
- Project lifecycle
- Crisis config

Service này **không** sở hữu bảng `data_sources` vật lý; chỉ dùng `project_id` logical reference khi tích hợp với ingest.

## Runtime API (Implemented)

Base group runtime: `/api/v1`

- `POST /campaigns`
- `GET /campaigns`
- `GET /campaigns/:id`
- `PUT /campaigns/:id`
- `DELETE /campaigns/:id`
- `POST /campaigns/:id/projects`
- `GET /campaigns/:id/projects`
- `GET /projects/:projectId`
- `PUT /projects/:projectId`
- `DELETE /projects/:projectId`
- `PUT /projects/:projectId/crisis-config`
- `GET /projects/:projectId/crisis-config`
- `DELETE /projects/:projectId/crisis-config`
- `GET /health`, `GET /ready`, `GET /live`
- `GET /swagger/*any`

## Integration Contracts (Planned)

- Kafka events: `project.activated`, `project.paused`, `project.resumed`, `project.archived`
- Internal API gọi ingest:
  - `PUT /ingest/datasources/{id}/crawl-mode`

## Deprecation

| Deprecated | Canonical |
|---|---|
| `/sources/*` | `/datasources/*` |
| `PUT /ingest/sources/{id}/crawl-mode` | `PUT /ingest/datasources/{id}/crawl-mode` |

## Notes

- Nếu có sai lệch giữa README và runtime, ưu tiên runtime code + swagger.
- Các flow event-driven liên service hiện được coi là `Planned` cho đến khi có producer/consumer nghiệp vụ trong code.
