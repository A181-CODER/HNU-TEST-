#!/usr/bin/env bash
set -Eeuo pipefail

# Usage: ./scripts/import_student_codes.sh /secure/path/student_codes.txt
# The input file is deliberately external and must not be committed.
INPUT_FILE="${1:-}"
if [[ -z "$INPUT_FILE" || ! -f "$INPUT_FILE" ]]; then
  echo "Usage: $0 /secure/path/student_codes.txt" >&2
  exit 1
fi

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"
PROJECT="${COMPOSE_PROJECT_NAME:-hnu-test}"
POSTGRES_DB="${POSTGRES_DB:-hnu_test}"
POSTGRES_USER="${POSTGRES_USER:-hnu}"
COMPOSE_ENV_ARGS=()
COMPOSE_CMD=(docker compose)
if ! docker info >/dev/null 2>&1; then
  if sudo docker info >/dev/null 2>&1; then
    COMPOSE_CMD=(sudo docker compose)
  else
    echo "Docker is not available to the current user." >&2
    exit 1
  fi
fi
if [[ -n "${ENV_FILE:-}" ]]; then
  COMPOSE_ENV_ARGS+=(--env-file "$ENV_FILE")
elif [[ -f .env ]]; then
  COMPOSE_ENV_ARGS+=(--env-file .env)
fi

TMP_FILE="$(mktemp)"
trap 'rm -f "$TMP_FILE"' EXIT

awk 'NF { gsub(/\r/, ""); if ($0 !~ /^921220[0-9]{3}$/) { print "Invalid student code: " $0 > "/dev/stderr"; bad=1 } else { print $0 } } END { if (bad) exit 1 }' "$INPUT_FILE" | sort -u > "$TMP_FILE"
COUNT="$(wc -l < "$TMP_FILE")"
if [[ "$COUNT" -eq 0 ]]; then
  echo "No valid student codes found." >&2
  exit 1
fi

# COPY receives only the validated code column. No names, emails, or passwords are imported.
printf 'Validated %s unique student codes. Importing into registry...\n' "$COUNT"
{
  printf 'BEGIN;\n'
  printf 'CREATE TEMP TABLE incoming_student_codes(student_number varchar(80));\n'
  printf 'COPY incoming_student_codes(student_number) FROM STDIN;\n'
  cat "$TMP_FILE"
  printf '\\.\n'
  printf "INSERT INTO student_identity_registry(student_number, status, source, updated_at) SELECT student_number, 'pending', 'controlled-import', now() FROM incoming_student_codes ON CONFLICT (student_number) DO UPDATE SET updated_at=now(), status=CASE WHEN student_identity_registry.status='disabled' THEN 'disabled' ELSE student_identity_registry.status END;\n"
  printf 'COMMIT;\n'
} | "${COMPOSE_CMD[@]}" --project-name "$PROJECT" "${COMPOSE_ENV_ARGS[@]}" -f "$COMPOSE_FILE" exec -T postgres psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"
printf 'Imported %s codes into student_identity_registry. They remain pending until linked to authenticated student users.\n' "$COUNT"
