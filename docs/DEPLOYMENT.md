# VPS Production Deployment

This repository is now **VPS-ready**, but it has not been deployed to a user-owned server because no VPS credentials or domain were provided. The production topology is defined in `docker-compose.prod.yml` and uses Caddy as the public reverse proxy.

> Do not run the production stack with the development `docker-compose.yml`. Do not commit `deploy/.env.production`; it is ignored by Git.

## Topology

```text
Internet
   │
   ├── 80/443 → Caddy (automatic HTTPS)
   │                 ├── /          → frontend (Nginx)
   │                 └── /api/*     → backend (Go + WebSocket)
   │                                   ├── postgres (private app network)
   │                                   └── python-proctoring (private app network)
   └── persistent volumes: Caddy certificates/config + PostgreSQL data
```

The `app` network is internal and PostgreSQL/Python are not published to the host. The `edge` network contains only Caddy, frontend and backend. WebSocket upgrade requests are passed through by Caddy using the same `/api/v1/proctoring/ws` path.

## Prerequisites

Use a fresh Ubuntu VPS with Docker Engine and the Docker Compose plugin. Point the domain's `A`/`AAAA` record to the VPS before starting Caddy. Allow inbound TCP ports 80 and 443; PostgreSQL port 5432 must remain closed to the public internet.

The VPS should have at least 2 vCPUs, 4 GB RAM and 40 GB of SSD for the current development-sized deployment. Production sizing, load testing, observability and external backup storage remain part of Phase 6.

## First deployment

Clone the repository on the VPS and create the private environment file:

```bash
git clone https://github.com/A181-CODER/HNU-TEST-.git
cd HNU-TEST-
cp deploy/.env.production.example deploy/.env.production
chmod 600 deploy/.env.production
```

Set the domain and generate URL-safe secrets. The database password must be written both to `POSTGRES_PASSWORD` and to `DATABASE_URL`:

```bash
DB_PASS="$(openssl rand -hex 32)"
JWT="$(openssl rand -hex 32)"
sed -i \
  -e "s#^DOMAIN=.*#DOMAIN=exam.example.edu#" \
  -e "s#^PUBLIC_BASE_URL=.*#PUBLIC_BASE_URL=https://exam.example.edu#" \
  -e "s#^POSTGRES_PASSWORD=.*#POSTGRES_PASSWORD=${DB_PASS}#" \
  -e "s#^DATABASE_URL=.*#DATABASE_URL=postgres://hnu:${DB_PASS}@postgres:5432/hnu_test?sslmode=disable#" \
  -e "s#^JWT_SECRET=.*#JWT_SECRET=${JWT}#" \
  deploy/.env.production
chmod 600 deploy/.env.production
```

Review the file without exposing it in logs, validate the rendered Compose configuration, and deploy:

```bash
docker compose --env-file deploy/.env.production -f docker-compose.prod.yml config >/tmp/hnu-compose-config.yml
./deploy/deploy.sh
curl -fsS https://exam.example.edu/health
curl -fsS https://exam.example.edu/ready
```

Caddy obtains and renews the certificate automatically after DNS resolves and ports 80/443 are reachable. The first certificate can take a short time; inspect `docker compose logs caddy` if HTTPS is not ready.

## Operations

Use the following commands from the repository root:

```bash
docker compose --env-file deploy/.env.production -f docker-compose.prod.yml ps
docker compose --env-file deploy/.env.production -f docker-compose.prod.yml logs -f backend
docker compose --env-file deploy/.env.production -f docker-compose.prod.yml logs -f caddy
```

Create a compressed PostgreSQL backup and keep the default fourteen-day local retention window:

```bash
./deploy/backup-postgres.sh
```

A production operator should schedule that script with a service account and copy the resulting files to encrypted off-host storage. A local backup on the same VPS is not sufficient disaster recovery.

## Firewall and backup minimums

The public firewall should allow SSH from a restricted administrative source and TCP 80/443 from the internet. Do not publish ports 5432, 8080 or 8000. Enable unattended security updates according to the VPS provider's policy, and test a database restore periodically on a separate environment.

## Migration warning

The current migration directory is mounted as PostgreSQL initialization input. Those files run automatically only when PostgreSQL initializes an empty data volume. On an existing production database, do not assume that adding a new file will apply it automatically. Phase 6 must introduce a versioned migration runner and an explicit maintenance procedure before frequent production schema changes.

## Secrets and privacy

Secrets belong only in `deploy/.env.production` or a VPS secret manager. Rotate `JWT_SECRET` deliberately because rotating it invalidates active access tokens. Replace every development seed password before an institutional deployment. Configure camera/privacy notices, event retention and backup encryption according to the institution's approved policy before enabling real student data.

## Current boundary

This commit prepares the repository for a VPS deployment; it does not claim that a real VPS, domain, DNS record, TLS certificate or production database has been provisioned. Final Production Hardening remains necessary for refresh-token rotation, password recovery and email verification, rate limiting, observability, external backups, migration versioning, load testing, self-hosted/pinned AI assets and disaster recovery.

## Development demo login password

For local demonstrations only, the seeded development accounts can be switched to the shared password `12345678` with `scripts/set-demo-password.sh`. The script refuses production-named environment files and production Compose files, updates only the clearly labelled `.local` demo accounts, and must never be run against the VPS production database. Replace all development credentials before institutional deployment.
