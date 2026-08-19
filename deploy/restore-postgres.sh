#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ROOT_DIR}/deploy/.env.production"
BACKUP_FILE="${1:-}"
COMPOSE_PROJECT_ARGS=()
[[ -n "${COMPOSE_PROJECT_NAME:-}" ]] && COMPOSE_PROJECT_ARGS=(-p "$COMPOSE_PROJECT_NAME")
COMPOSE=(docker compose "${COMPOSE_PROJECT_ARGS[@]}" --env-file "$ENV_FILE" -f "${ROOT_DIR}/docker-compose.prod.yml")

[[ -f "$ENV_FILE" ]] || { echo "Missing $ENV_FILE" >&2; exit 1; }
# The operator owns this private file; load only its environment assignments.
set -a
. "$ENV_FILE"
set +a
[[ -f "$BACKUP_FILE" ]] || { echo "Usage: $0 backups/hnu_test_YYYYMMDDTHHMMSSZ.sql.gz" >&2; exit 1; }
[[ "${CONFIRM_RESTORE:-}" == "YES" ]] || { echo "Set CONFIRM_RESTORE=YES to replace database contents." >&2; exit 1; }

"${COMPOSE[@]}" exec -T -e PGPASSWORD="${POSTGRES_PASSWORD}" postgres \
  psql -v ON_ERROR_STOP=1 -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" \
  < <(gzip -dc "$BACKUP_FILE")

echo "Restore completed from $BACKUP_FILE"
