# University Management and Resource Authorization

## Scope

Phase 4 introduces the institutional resource model used by HNU TEST:

> University → Faculty → Department → Course → Users, Exams, Attempts, Results and Proctoring

The model is additive. It preserves the existing exam lifecycle and realtime proctoring services while making the backend resolve a resource scope before returning or mutating protected data.

## Data model

The migration `database/migrations/006_university_scope.sql` adds three assignment surfaces. `resource_memberships` assigns `university_admin`, `instructor` or `proctor` to a university, faculty, department or course scope. `course_students` enrolls students in courses and `course_instructors` assigns teaching staff. `exam_proctors` is the narrow assignment that gives a proctor access to a particular exam and its attempts.

A membership at a lower level is inherited upward for directory visibility. A course membership can therefore reveal its parent department, faculty and university in `/organization/tree`, but it cannot reveal a sibling course. A proctor's live monitoring access remains narrower: it requires a direct row in `exam_proctors`.

## Authorization matrix

| Actor | University directory | Course | Exam | Attempt/result | Proctoring |
|---|---|---|---|---|---|
| `super_admin` | All | All | All | All | All |
| `university_admin` | Assigned hierarchy | Assigned hierarchy | Assigned course scope | Assigned exam scope | Assigned exam scope |
| `instructor` | Assigned hierarchy | Assigned course scope | Assigned course scope | Assigned exam scope | Assigned exam scope |
| `proctor` | Assigned hierarchy | Not a teaching scope | Directly assigned exams | Directly assigned exam attempts | Directly assigned exams only |
| `student` | Enrolled course ancestry | Enrolled courses | Enrolled courses | Own attempt only | Own attempt signals only |

The checks are performed in Go helpers in `backend-go/internal/httpapi/scope.go`. The React dashboard uses the same API but is not relied on as a security control.

## API surface

`GET /api/v1/organization/tree` returns the caller's filtered hierarchy and the caller's roles. `GET /api/v1/organization/overview` returns counts within that scope. Super administrators and university administrators can create scoped memberships with `POST /api/v1/organization/assignments`. Scoped instructors and administrators can enroll students, assign instructors and assign proctors through the course and exam assignment routes documented in `docs/API.md` and `docs/openapi.yaml`.

Every exam must resolve to a course. The legacy `courseCode` request remains accepted for compatibility and is resolved against `courses.code`; new clients should send `courseId`. Exam creation, read, draft mutation, publish, scheduling, results and proctoring use the exam's course to enforce scope. The student list additionally requires `course_students`, and the proctor list/dashboard/WebSocket require direct `exam_proctors` assignment.

## Verification

The repeatable fixture `scripts/phase4_isolation_fixture.sql` creates a second university, course, instructor, proctor, student and exam. `scripts/phase4_isolation_e2e.py` verifies that University A does not see University B, an instructor cannot read or create against an out-of-scope course, a proctor cannot read or open a WebSocket for an unassigned exam, and a student cannot access another student's attempt. The same test verifies that a super administrator can see both university scopes.

## Known limitations

Phase 4 establishes the authorization foundation but does not yet provide full CRUD for faculties, departments and courses, bulk SIS/SCIM synchronization, delegated administration workflows, or a complete user-directory management screen. Resource scope is currently maintained by authenticated administration API calls and migrations. Analytics, exports, scheduled reports and advanced institutional configuration are intentionally deferred to the next phase.
