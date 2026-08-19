# Student Identity and Instructor Workspace

## Student codes

Student codes are treated as identity identifiers, not passwords. The repository contains the migration and import utility only; real codes must be supplied from a deployment-local file and must never be committed to GitHub, GitHub Pages, logs, or the frontend bundle.

Run the controlled import against the selected database with:

```bash
POSTGRES_DB=hnu_test POSTGRES_USER=hnu \
  ./scripts/import_student_codes.sh /secure/path/student_codes.txt
```

For production, use the production compose file and environment file explicitly:

```bash
COMPOSE_FILE=docker-compose.prod.yml \
ENV_FILE=deploy/.env.production \
COMPOSE_PROJECT_NAME=hnu-production \
./scripts/import_student_codes.sh /secure/path/student_codes.txt
```

The utility validates the current HNU student-code format, removes duplicates, and inserts only the code into `student_identity_registry` with `pending` status. A code becomes usable only after an administrator links it to an authenticated student account through `POST /api/v1/organization/student-identities/link`. The student must still authenticate with a password; possession of a code alone does not grant access.

Before an exam attempt is created, the backend requires a linked and verified student identity. The student portal asks for the code and calls `POST /api/v1/student/verify-code`. The backend checks both the authenticated user and the registry linkage. A mismatch is rejected with `403`; an unlinked code is rejected with `409`.

## Instructor workspace

The instructor workspace is available from the **صفحة الدكتور / Instructor workspace** button in the administration header or from the **الاختبارات / Exams** navigation item. The workspace uses the existing instructor authentication and resource-scope authorization.

The current authoring flow is:

1. Sign in as an instructor.
2. Create an examination draft by course code.
3. Add a multiple-choice question and attach it to the draft.
4. Publish the examination. The API refuses publication without at least one attached question.
5. Schedule it through the existing scheduling endpoint.

The backend routes are `POST /api/v1/exams`, `POST /api/v1/questions`, `POST /api/v1/exams/{id}/questions`, `POST /api/v1/exams/{id}/publish`, and `POST /api/v1/exams/{id}/schedule`. Course and exam scope checks remain enforced in the backend.

## Production privacy boundary

The GitHub Pages site is a static interface preview. It must not receive real student codes or authenticated examination data. Real identity verification, PostgreSQL persistence, camera events, Python analysis, and WebSocket monitoring require the VPS deployment with protected environment variables and access-controlled database operations.
