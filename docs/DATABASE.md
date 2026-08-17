# Database

`database/migrations/001_initial.sql` creates the normalized PostgreSQL foundation. It includes users, roles, permissions, institutional hierarchy, courses, question banks, typed questions and options, exams, schedules, sessions, attempts, answers, results, proctoring sessions/events, audit logs, notifications, announcements, refresh sessions, and system settings.

Sensitive values are never stored in plaintext. Passwords are bcrypt hashes. Refresh token storage is represented by token-hash columns and must be written only after hashing. The migration includes foreign keys, uniqueness, check constraints and indexes for operational queries.

For production, use a migration runner with version tracking rather than mounting raw SQL as an init directory. Apply retention policies for audit and proctoring records based on institutional policy.
