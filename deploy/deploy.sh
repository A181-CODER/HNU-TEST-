#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ROOT_DIR}/deploy/.env.production"
COMPOSE_PROJECT_ARGS=()
[[ -n "${COMPOSE_PROJECT_NAME:-}" ]] && COMPOSE_PROJECT_ARGS=(-p "$COMPOSE_PROJECT_NAME")
COMPOSE=(docker compose "${COMPOSE_PROJECT_ARGS[@]}" --env-file "$ENV_FILE" -f "${ROOT_DIR}/docker-compose.prod.yml")

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE. Copy deploy/.env.production.example and fill it on the VPS." >&2
  exit 1
fi
command -v docker >/dev/null || { echo "Docker is required" >&2; exit 1; }
docker compose version >/dev/null || { echo "Docker Compose plugin is required" >&2; exit 1; }

DOMAIN="$(awk -F= '$1 == "DOMAIN" { print substr($0, index($0, "=") + 1); exit }' "$ENV_FILE")"
[[ -n "$DOMAIN" ]] || { echo "DOMAIN is missing in $ENV_FILE" >&2; exit 1; }

cd "$ROOT_DIR"
"${COMPOSE[@]}" config >/dev/null
"${COMPOSE[@]}" pull postgres caddy
"${COMPOSE[@]}" up -d --build --remove-orphans
"${COMPOSE[@]}" ps

echo "Deployment started. Verify HTTPS and readiness with:"
echo "  curl -fsS https://${DOMAIN}/health"
echo "  curl -fsS https://${DOMAIN}/ready"
