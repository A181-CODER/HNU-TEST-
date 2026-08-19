#!/usr/bin/env bash
set -Eeuo pipefail

API="${API:-http://localhost:8080/api/v1}"
PASSWORD="${PASSWORD:-ChangeMe-Development-Only!}"
login() {
  curl -fsS "$API/auth/login" -H 'Content-Type: application/json' \
    --data "{\"email\":\"$1\",\"password\":\"$PASSWORD\"}" | jq -r .accessToken
}

student_token="$(login student@hnu-test.local)"
instructor_token="$(login instructor@hnu-test.local)"
identity="$(curl -fsS "$API/student/identity" -H "Authorization: Bearer $student_token")"
verified="$(curl -fsS "$API/student/verify-code" -H "Authorization: Bearer $student_token" -H 'Content-Type: application/json' --data '{"studentCode":"DEMO-2026-001"}')"
wrong_status="$(curl -sS -o /tmp/hnu-identity-wrong-code.json -w '%{http_code}' "$API/student/verify-code" -H "Authorization: Bearer $student_token" -H 'Content-Type: application/json' --data '{"studentCode":"921220001"}')"
exams_status="$(curl -sS -o /tmp/hnu-instructor-exams.json -w '%{http_code}' "$API/exams" -H "Authorization: Bearer $instructor_token")"

[[ "$(jq -r .verified <<<"$identity")" == true ]]
[[ "$(jq -r .verified <<<"$verified")" == true ]]
[[ "$wrong_status" == 403 ]]
[[ "$exams_status" == 200 ]]
printf '%s\n' 'IDENTITY_INSTRUCTOR_E2E_PASS'
