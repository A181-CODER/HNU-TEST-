# HNU TEST

**HNU TEST** is a secure, bilingual online examination and human-reviewed proctoring platform branded by **KING ABDO**. It is designed for universities and educational institutions without claiming university ownership or endorsement.

> جميع الحقوق محفوظة لـ KING ABDO © 2026  
> All Rights Reserved © KING ABDO 2026

## Current delivery

This repository contains a working foundation rather than a fake dashboard: a Go API with PostgreSQL-ready migrations, secure password authentication, RBAC, university resource hierarchy and scope authorization, exam lifecycle endpoints, server-side grading, audit logging, a Python/FastAPI event-analysis service, a realtime WebSocket proctoring service, and a React/TypeScript interface that consumes the API. Features that require production infrastructure, such as durable email delivery, Redis-backed distributed rate limits, and browser camera inference with a trained model, are represented by explicit interfaces and documented as partial until configured.

## Run locally

Requirements: Docker Compose, Go 1.22+, Python 3.11+, Node 20+ and npm/pnpm.

```bash
cp .env.example .env
docker compose up --build
```

The web application is available at `http://localhost:5173`, the API at `http://localhost:8080`, the API health endpoint at `http://localhost:8080/health`, and the proctoring service at `http://localhost:8000/health`.

For native development:

```bash
cd backend-go && go test ./... && go run ./cmd/api
cd frontend && npm install && npm run dev
cd ai-proctoring-python && python -m venv .venv && . .venv/bin/activate && pip install -r requirements.txt && uvicorn app.main:app --reload --port 8000
```

The seed command creates clearly labelled development accounts. Passwords are only shown in the local development documentation and must not be reused in production.

## VPS-ready production deployment

The repository includes a production-oriented `docker-compose.prod.yml` with Caddy automatic HTTPS, WebSocket pass-through, private application networking, PostgreSQL persistence, restart policies, health checks, and environment-based secrets. It is intentionally not deployed to a real server yet because no VPS or domain credentials have been provided.

Start from [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md), copy `deploy/.env.production.example` to the untracked `deploy/.env.production`, fill in a real domain and generated secrets, then run `./deploy/deploy.sh`. Backup and guarded restore scripts are available under `deploy/`.

### GitHub Pages frontend preview

The frontend is also published automatically at [a181-coder.github.io/HNU-TEST-](https://a181-coder.github.io/HNU-TEST-/) through GitHub Actions. This public Pages site is a static demonstration of the KING ABDO interface and uses demo data when no API URL is configured. The complete authenticated examination flow, PostgreSQL persistence, camera event capture, Python analysis and realtime WebSocket proctoring require the VPS deployment described above.

## Documentation

- [Architecture](docs/ARCHITECTURE.md)
- [API and OpenAPI](docs/API.md)
- [Database](docs/DATABASE.md)
- [Organization and resource authorization](docs/ORGANIZATION.md)
- [Security](docs/SECURITY.md)
- [Proctoring and privacy](docs/PROCTORING.md)
- [Deployment](docs/DEPLOYMENT.md)
- [Testing and QA](docs/TESTING.md)
- [Environment](docs/ENVIRONMENT.md)
- [Contributing](docs/CONTRIBUTING.md)

## License

The repository includes the Apache-2.0 license inherited from the initial repository. Review licensing, institutional agreements, privacy notices, and retention obligations with qualified counsel before a production deployment.
