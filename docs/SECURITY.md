# Security

The API applies password hashing, short-lived signed access tokens, RBAC checks, strict JSON responses, CORS configuration, no-store responses, secure response headers, validation at the handler boundary, and audit logging for successful login. Passwords, tokens, and secrets must not enter logs.

Before production, replace the development JWT secret, terminate TLS at a hardened reverse proxy, use secure HTTP-only refresh cookies with rotation and revocation, add CSRF protection for cookie-authenticated browser flows, add a shared rate limiter, add account lockout persistence, configure secret management, run dependency and container scanning, and commission an independent penetration test.

The browser camera is permission-based and event-oriented. The platform must explain what is processed, why it is processed, what is retained, who reviews it, and how deletion or retention requests are handled. An event confidence score is not a finding of misconduct.
