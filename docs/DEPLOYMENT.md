# Deployment

Build the stack with `docker compose build` and run it with `docker compose up -d`. Put a TLS reverse proxy in front of the frontend and API, restrict PostgreSQL to a private network, provide secrets through the deployment secret manager, and configure daily encrypted backups with restore drills.

Health probes are `/health` for liveness and `/ready` for readiness. Add structured log shipping, database metrics, API latency/error metrics, WebSocket connection metrics when realtime monitoring is enabled, and alerting for failed backups, authentication spikes and proctoring-service outages.

The current compose file is a development baseline. It must not be treated as a production deployment without TLS, secret rotation, non-default credentials, network segmentation, resource limits, migration versioning, backups, monitoring and an incident response runbook.
