#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
ENV_FILE="${ENV_FILE:-.env}"
PROJECT_NAME="${COMPOSE_PROJECT_NAME:-hnu-test-}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"

if [[ "$ENV_FILE" == *production* || "$COMPOSE_FILE" == *prod* || "${APP_ENV:-development}" == "production" ]]; then
  echo "Refusing to set the demo password in production." >&2
  exit 1
fi

if [[ ! -f "$ENV_FILE" && "$ENV_FILE" != ".env" ]]; then
  echo "Missing environment file: $ENV_FILE" >&2
  exit 1
fi

COMPOSE=(docker compose --project-name "$PROJECT_NAME")
if [[ "${USE_SUDO:-1}" == "1" ]] && ! docker info >/dev/null 2>&1; then
  COMPOSE=(sudo docker compose --project-name "$PROJECT_NAME")
fi

DB_USER="${POSTGRES_USER:-hnu}"
DB_NAME="${POSTGRES_DB:-hnu_test}"
HASH='$2a$10$yhifFE59061fr7/qIy5JeOYpnPtDabrc5GRGfz70VYOOcTqA7fuvm'

"${COMPOSE[@]}" -f "$COMPOSE_FILE" exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -v password_hash="$HASH" <<'SQL'
UPDATE users
SET password_hash = :'password_hash', failed_login_count = 0, locked_until = NULL, updated_at = now()
WHERE email IN (
  'admin@hnu-test.local',
  'instructor@hnu-test.local',
  'proctor@hnu-test.local',
  'student@hnu-test.local',
  'instructor-b@hnu-test.local',
  'proctor-b@hnu-test.local',
  'student-b@hnu-test.local'
);
SQL
printf '%s\n' 'Development demo accounts now accept password 12345678. Do not reuse it in production.'
