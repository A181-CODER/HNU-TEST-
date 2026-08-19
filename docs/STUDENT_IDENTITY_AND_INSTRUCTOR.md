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

## Bulk administration workspace

Authorized administrators can open **إدارة الجامعة → ربط أكواد الطلاب**. The workspace loads the registry status and authenticated student candidates, shows total/pending/linked counts, supports direct code-to-account selection, accepts reviewed `studentCode,UserID` rows, displays a removable preview, and commits the selected mappings in one transaction. The backend endpoints are `GET /api/v1/organization/student-identities`, `GET /api/v1/organization/student-identities/candidates`, and `POST /api/v1/organization/student-identities/bulk-link`. All three are restricted to `super_admin` and `university_admin`; student and instructor tokens receive `403`.

A failed row prevents the entire bulk request from being committed. The backend rejects invalid codes, non-student accounts, already-owned codes, and cross-account conflicts. Successful operations create an audit event. The public GitHub Pages build includes the interface shell but has no API URL and no real student data; the authenticated VPS deployment is required to load and mutate the registry.

## Instructor dashboard and live monitoring

The instructor workspace now exposes three operational actions from the same authenticated page: create a draft examination, add validated multiple-choice questions with an explicitly selected correct option, and publish/schedule the examination through the existing scoped API. The schedule form calls `POST /api/v1/exams/{id}/schedule` with a start and end window.

The **متابعة الطلاب حياً** action opens the existing human-review control room using the instructor token. It loads scoped active attempts, displays connection and camera/face-count status, receives proctoring events over WebSocket, calculates visible risk categories, and provides review and dismiss actions for open events. The monitoring view receives technical signals and event metadata; it does not expose an unnecessary raw-video archive.

Student flow now performs the identity-code check and then opens a pre-exam camera-readiness dialog before creating the exam attempt. Once the attempt begins, the secure exam room requests the webcam again, runs browser-side face detection, sends camera/face-count signals, and records camera interruption, tab, fullscreen, focus, and network events for human review.
