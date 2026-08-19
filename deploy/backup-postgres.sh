#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ROOT_DIR}/deploy/.env.production"
BACKUP_DIR="${BACKUP_DIR:-${ROOT_DIR}/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-14}"
COMPOSE_PROJECT_ARGS=()
[[ -n "${COMPOSE_PROJECT_NAME:-}" ]] && COMPOSE_PROJECT_ARGS=(-p "$COMPOSE_PROJECT_NAME")
COMPOSE=(docker compose "${COMPOSE_PROJECT_ARGS[@]}" --env-file "$ENV_FILE" -f "${ROOT_DIR}/docker-compose.prod.yml")

[[ -f "$ENV_FILE" ]] || { echo "Missing $ENV_FILE" >&2; exit 1; }
# The operator owns this private file; load only its environment assignments.
set -a
. "$ENV_FILE"
set +a
mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUT="${BACKUP_DIR}/hnu_test_${STAMP}.sql.gz"

"${COMPOSE[@]}" exec -T -e PGPASSWORD="${POSTGRES_PASSWORD}" postgres \
  pg_dump --format=plain --no-owner --no-privileges -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" \
  | gzip -9 > "$OUT"

find "$BACKUP_DIR" -type f -name 'hnu_test_*.sql.gz' -mtime "+${RETENTION_DAYS}" -delete
chmod 600 "$OUT"
echo "Created $OUT"
