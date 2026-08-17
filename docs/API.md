# API

The API exposes the real exam lifecycle and the Phase 3 realtime proctoring flow. Protected HTTP endpoints require `Authorization: Bearer <access-token>`. The server owns schedule validation, attempt limits, expiration, answer persistence, randomization, grading, result visibility and monitoring risk state.

| Method | Path | Access | Purpose |
|---|---|---|---|
| GET | `/health` | Public | Liveness |
| GET | `/ready` | Public | PostgreSQL readiness |
| POST | `/api/v1/auth/login` | Public | Password login and short-lived access token |
| GET | `/api/v1/me` | Authenticated | Current identity and roles |
| GET/POST | `/api/v1/exams` | Instructor/admin | List or create exams |
| GET/PATCH | `/api/v1/exams/{id}` | Auth/instructor | Read or update a draft |
| POST | `/api/v1/exams/{id}/questions` | Instructor/admin | Attach a question to a draft |
| POST | `/api/v1/exams/{id}/publish` | Instructor/admin | Publish a draft containing questions |
| POST | `/api/v1/exams/{id}/schedule` | Instructor/admin | Set server-validated schedule |
| GET/POST | `/api/v1/questions` | Instructor/admin | List or create questions |
| GET | `/api/v1/student/exams` | Student | Authorized exam list with availability and attempts |
| POST | `/api/v1/exams/{id}/start` | Student | Verify access, create/resume attempt and question set |
| GET | `/api/v1/attempts/{id}` | Owner/proctor/instructor | Load attempt, answers and server deadline |
| POST/PATCH | `/api/v1/attempts/{id}/answers/{questionId}` | Attempt owner | Validate and autosave an answer |
| POST | `/api/v1/attempts/{id}/submit` | Attempt owner/instructor | Freeze, grade and generate result transactionally |
| GET | `/api/v1/attempts/{id}/result` | Owner/instructor | Read result subject to publication policy |
| POST | `/api/v1/attempts/{id}/proctoring-events` | Attempt owner/proctor | Persist browser lifecycle event and broadcast it |
| POST | `/api/v1/attempts/{id}/proctoring-signal` | Attempt owner/proctor | Analyze derived face/camera/tab/fullscreen/network signal through Python |
| GET | `/api/v1/proctoring/active-attempts` | Proctor/instructor/admin | Live sessions, connection state, face count, risk and open events |
| GET | `/api/v1/proctoring/attempts/{id}/events` | Proctor/instructor/admin | Recent monitoring events for an attempt |
| POST | `/api/v1/proctoring/events/{eventId}/review` | Proctor/instructor/admin | Record `confirmed`, `dismissed` or `needs_followup` with a note |
| GET | `/api/v1/proctoring/ws?token=...` | Proctor/instructor/admin | Authenticated WebSocket event feed |
| GET | `/api/v1/instructor/exams/{id}/results` | Instructor/admin/proctor | Review exam results |
| POST | `/api/v1/results/{id}/publish` | Instructor/admin | Publish a result to the student |

## Monitoring event model

Every persisted monitoring event has `eventType`, `severity`, `riskScore`, `confidence`, `source`, `occurredAt`, `reviewStatus` and privacy-conscious `metadata`. The risk score prioritizes human attention; it is not a cheating verdict. Session state also tracks `connectionStatus`, `monitoringStatus`, `lastHeartbeatAt`, `lastSignalAt` and `lastFaceCount`.

## Lifecycle states

Exams move through `draft`, `published`, `scheduled`, `active`, `ended` and the legacy `closed`/`archived` states. Attempts use `in_progress`, `submitted`, `auto_submitted` and `expired`. The expiration timestamp is persisted on the attempt and session and is never accepted from the browser.

## Request examples

```json
{
  "title": "Software Engineering Midterm",
  "courseCode": "SWE-201",
  "durationMinutes": 30,
  "attemptLimit": 1,
  "passingScore": 60,
  "totalMarks": 20,
  "negativeMarking": 0.25,
  "randomizeQuestions": true,
  "randomizeOptions": true,
  "allowReview": true,
  "resultVisibility": "not_published",
  "instructions": "Read each question carefully."
}
```

A derived proctoring signal uses explicit browser state and face count:

```json
{
  "attemptId": "attempt-uuid",
  "faceCount": 2,
  "cameraAvailable": true,
  "tabVisible": true,
  "fullscreen": true,
  "networkOnline": true,
  "networkRttMs": 42,
  "metadata": {"source": "mediapipe"}
}
```

An answer is persisted as logical option identities, not display positions:

```json
{
  "questionId": "60000000-0000-0000-0000-000000000001",
  "values": ["A"],
  "textAnswer": "",
  "markedForReview": true
}
```

OpenAPI is maintained in `docs/openapi.yaml`. The OpenAPI file documents the stable public surface; update it whenever a new route or request schema is introduced.
