#!/usr/bin/env bash
set -euo pipefail
API=${API:-http://localhost:8080/api/v1}
login(){ curl -fsS -X POST "$API/auth/login" -H 'Content-Type: application/json' -d "$1"; }
INSTRUCTOR=$(login '{"email":"instructor@hnu-test.local","password":"ChangeMe-Development-Only!"}')
ITOKEN=$(printf '%s' "$INSTRUCTOR" | python3 -c 'import json,sys; print(json.load(sys.stdin)["accessToken"])')
EXAM=$(curl -fsS -X POST "$API/exams" -H "Authorization: Bearer $ITOKEN" -H 'Content-Type: application/json' -d '{"title":"E2E Lifecycle Examination","description":"Acceptance flow","courseCode":"SWE-201","durationMinutes":30,"attemptLimit":1,"passingScore":60,"totalMarks":2,"negativeMarking":0.5,"randomizeQuestions":true,"randomizeOptions":true,"allowReview":true,"resultVisibility":"not_published","instructions":"Read carefully"}')
EID=$(printf '%s' "$EXAM" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
curl -fsS -X POST "$API/exams/$EID/questions" -H "Authorization: Bearer $ITOKEN" -H 'Content-Type: application/json' -d '{"questionId":"60000000-0000-0000-0000-000000000001","position":1,"points":2}' >/tmp/hnu-attach.json
curl -fsS -X POST "$API/exams/$EID/publish" -H "Authorization: Bearer $ITOKEN" >/tmp/hnu-publish.json
START=$(date -u -d '1 minute ago' +%Y-%m-%dT%H:%M:%SZ)
END=$(date -u -d '30 minutes' +%Y-%m-%dT%H:%M:%SZ)
curl -fsS -X POST "$API/exams/$EID/schedule" -H "Authorization: Bearer $ITOKEN" -H 'Content-Type: application/json' -d "{\"startAt\":\"$START\",\"endAt\":\"$END\"}" >/tmp/hnu-schedule.json
STUDENT=$(login '{"email":"student@hnu-test.local","password":"ChangeMe-Development-Only!"}')
STOKEN=$(printf '%s' "$STUDENT" | python3 -c 'import json,sys; print(json.load(sys.stdin)["accessToken"])')
STUDENT_EXAMS=$(curl -fsS "$API/student/exams" -H "Authorization: Bearer $STOKEN")
ATTEMPT=$(curl -fsS -X POST "$API/exams/$EID/start" -H "Authorization: Bearer $STOKEN")
AID=$(printf '%s' "$ATTEMPT" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
curl -fsS -X PATCH "$API/attempts/$AID/answers/60000000-0000-0000-0000-000000000001" -H "Authorization: Bearer $STOKEN" -H 'Content-Type: application/json' -d '{"questionId":"60000000-0000-0000-0000-000000000001","values":["A"],"markedForReview":true}' >/tmp/hnu-answer.json
SUBMISSION=$(curl -fsS -X POST "$API/attempts/$AID/submit" -H "Authorization: Bearer $STOKEN")
RID=$(printf '%s' "$SUBMISSION" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
curl -fsS -X POST "$API/results/$RID/publish" -H "Authorization: Bearer $ITOKEN" >/tmp/hnu-publish-result.json
RESULT=$(curl -fsS "$API/attempts/$AID/result" -H "Authorization: Bearer $STOKEN")
printf '%s\n' "$STUDENT_EXAMS"
printf '%s\n' "$ATTEMPT"
printf '%s\n' "$SUBMISSION"
printf '%s\n' "$RESULT"
printf 'E2E_PASS exam=%s attempt=%s result=%s\n' "$EID" "$AID" "$RID"
