# API

The current HTTP surface is intentionally small and real. All protected endpoints require `Authorization: Bearer <access-token>` and return JSON errors without stack traces.

| Method | Path | Access | Purpose |
|---|---|---|---|
| GET | `/health` | Public | Liveness |
| GET | `/ready` | Public | Database readiness |
| POST | `/api/v1/auth/login` | Public | Password login and short-lived access token |
| GET | `/api/v1/me` | Authenticated | Current identity and roles |
| GET | `/api/v1/exams` | Authenticated | List non-deleted exams |
| POST | `/api/v1/exams` | Admin/instructor | Create a draft exam |
| POST | `/api/v1/exams/{id}/start` | Authenticated | Reserved exam session boundary |
| POST | `/api/v1/attempts/{id}/answers` | Authenticated | Reserved autosave boundary |
| POST | `/api/v1/attempts/{id}/submit` | Authenticated | Reserved server grading boundary |

OpenAPI is maintained in `docs/openapi.yaml`. The reserved endpoints return `501` rather than fake success until their transaction and integration tests are implemented.
