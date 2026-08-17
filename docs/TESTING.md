# Testing and QA

The Go service includes unit tests for password hashing, JWT claim preservation, minimum password length and objective grading including multiple-select and negative marking.

The acceptance path remains partial in this first commit: database-backed login and exam creation are implemented, while start, autosave and submit return an explicit `501` until their transactional exam-engine tests are added. QA must cover authentication, RBAC, scheduling, timer authority, autosave/reconnect conflict handling, camera permission, human review, result publication, CSV/JSON/PDF exports, RTL, accessibility, mobile layouts, security headers, rate limits, backups and concurrent sessions.

Run:

```bash
cd backend-go
go test ./...
```
