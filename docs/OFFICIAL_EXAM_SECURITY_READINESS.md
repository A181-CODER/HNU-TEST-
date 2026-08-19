# Official Examination Security Readiness

This document is the go/no-go checklist for moving HNU TEST from a controlled pilot to official university examinations. The system must not be treated as production-ready merely because the frontend is visible on GitHub Pages. The authenticated examination service, PostgreSQL database, camera events, Python analysis, and WebSocket monitoring belong on the protected VPS deployment.

## Current security baseline

The current platform already provides server-authoritative exam timing, authenticated roles, resource-level authorization, course and exam scoping, autosave, proctoring events, browser camera checks, face-count signals, fullscreen and tab events, network events, human review, PostgreSQL persistence, Caddy HTTPS/WebSocket routing, and a controlled student-code registry. The bulk identity workspace now adds an administrator-only preview-and-commit workflow.

| Control | Current state | Official-exam requirement |
|---|---|---|
| Student code is not a password | **Complete** | Keep this rule and communicate it to students |
| Codes imported outside Git | **Complete** | Use a protected file on the VPS and delete it after verified import |
| Pending-to-linked workflow | **Complete** | Link only after confirming the authoritative student roster |
| Bulk linking transaction | **Complete** | Review the preview and record the operator before commit |
| Backend identity check before attempt | **Complete** | Keep enabled; never rely only on frontend validation |
| Instructor resource scope | **Complete** | Test each faculty/department boundary before each exam period |
| Server-authoritative timer | **Complete** | Verify clock synchronization and timezone policy |
| Camera and browser-event capture | **Complete foundation** | Run a device and browser pilot before the official window |
| HTTPS and WebSocket routing | **Complete in production configuration** | Deploy only with a real domain and valid certificates |
| MFA for administrators, instructors, proctors | **Pending** | Must be implemented before official high-stakes exams |
| Rate limiting and abuse protection | **Pending** | Must be enabled at the reverse proxy and API layers |
| External encrypted backups | **Pending** | Must be configured and restored successfully before launch |
| Central monitoring and alerting | **Pending** | Must be available during every exam session |
| Formal retention and deletion policy | **Pending** | Must be approved by the university before collecting real events |

## Student identity operating procedure

The administrator should first export the authoritative roster from the university system and keep the source file in a protected location outside the repository. The import utility validates the configured code format, removes duplicates, and inserts the codes with `pending` status. The source file should be removed or encrypted after the database count and checksum have been verified.

The administrator then opens **إدارة الجامعة → ربط أكواد الطلاب**. The interface shows total, pending, and linked counts; supports selecting a pending code and an authenticated student account; accepts a reviewed `studentCode,UserID` mapping; displays a preview; and commits all rows in a single transaction. If one row fails, the transaction is rolled back. The operator must verify the preview against the authoritative roster before pressing **اعتماد الربط الجماعي**.

The system must not place real codes in GitHub, GitHub Pages, browser bundles, screenshots, support tickets, analytics, or application logs. A student still needs a normal authenticated account and password. A code alone must never grant access. Before an attempt is created, the backend verifies that the code and authenticated account are linked and that the student is enrolled in the course scope.

## Pre-exam technical gates

Before the official examination window, the operations team should deploy from a tagged commit to the VPS, populate a production-only environment file, generate a strong JWT secret and database password, configure the real domain in Caddy, and confirm that HTTP redirects to HTTPS. The PostgreSQL service must remain on the private Docker network, and only Caddy should be reachable from the public interface.

The team should then execute a controlled pilot with representative student, instructor, proctor, and administrator accounts. The pilot must cover login, identity verification, camera permission, face detection, camera interruption, tab visibility, fullscreen exit, network loss and recovery, autosave, resume, submission, grading, result publication, proctor WebSocket visibility, and human review. The pilot must include the weakest supported browser and network conditions expected on campus.

| Gate | Evidence required before go-live |
|---|---|
| Identity roster | Database count equals the approved roster; pending and linked counts are explained |
| Authorization | Cross-university and cross-course negative tests pass |
| Exam content | Instructor preview approved; publication and schedule verified |
| Camera readiness | Pilot evidence for supported browsers, permissions, and device failures |
| Proctoring | Proctor sees active attempts and high-severity events in real time |
| Recovery | Autosave, resume, submission, and timeout behavior verified |
| Backup | Fresh backup exists, checksum verified, and restore drill completed |
| Incident response | Named on-call staff, escalation channel, and student support procedure |
| Privacy | Consent, retention period, access list, and deletion procedure approved |
| Change freeze | No unreviewed code, schema, or configuration changes during the exam |

## Data protection requirements

Production secrets must be stored only in the VPS environment file or an approved secret manager, with restrictive file permissions. Database backups must be encrypted before leaving the VPS and retained according to the approved university policy. Access to student identity data must be limited to named university administrators; instructors and proctors should receive only the minimum information required for their scope and role.

Proctoring should collect the minimum technical signals needed for human review. The current browser design sends event metadata and face-count signals rather than an unnecessary raw-video archive. The university must still define whether any snapshots, logs, or event metadata are retained, for how long, who can review them, and how a student can challenge a decision. These decisions must be documented before official collection begins.

## Remaining blockers before official use

The project is **not yet cleared for official high-stakes examinations** until MFA, rate limiting, encrypted off-host backups with a successful restore drill, centralized monitoring, production migration discipline, and a university-approved privacy and retention policy are completed. The current GitHub Pages site is only a frontend demonstration and must not be used as the authenticated examination endpoint.

The next recommended sequence is to complete Phase 6 production hardening, deploy a private VPS staging environment, run a limited pilot with synthetic or approved test accounts, perform an independent security review, and only then conduct a controlled official examination with an on-call operations team.
