#!/usr/bin/env bash
set -Eeuo pipefail
BASE="${API_BASE:-http://localhost:8080/api/v1}"
PASSWORD='ChangeMe-Development-Only!'
login(){ curl -fsS "$BASE/auth/login" -H 'Content-Type: application/json' --data "$(jq -nc --arg email "$1" --arg password "$PASSWORD" '{email:$email,password:$password}')" | jq -r .accessToken; }
TOKEN="$(login instructor@hnu-test.local)"
[ -n "$TOKEN" ] && [ "$TOKEN" != null ]
EXAM="$(curl -fsS "$BASE/exams" -H "Authorization: Bearer $TOKEN")"
EXAM_ID="$(curl -fsS "$BASE/exams" -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' --data "$(jq -nc --arg title "Instructor E2E $(date -u +%Y%m%d%H%M%S)" '{title:$title,courseCode:"SWE-201",durationMinutes:45,instructions:"Controlled instructor dashboard acceptance test",passingScore:50,attemptLimit:1,totalMarks:1,negativeMarking:0,randomizeQuestions:false,randomizeOptions:false,allowReview:true,resultVisibility:"not_published"}')" | jq -r .id)"
[ -n "$EXAM_ID" ] && [ "$EXAM_ID" != null ]
QUESTION_ID="$(curl -fsS "$BASE/questions" -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' --data '{"type":"multiple_choice","prompt":"Controlled dashboard acceptance question","points":1,"difficulty":"medium","options":[{"key":"A","text":"Correct answer","isCorrect":true},{"key":"B","text":"Incorrect answer","isCorrect":false}]}' | jq -r .id)"
[ -n "$QUESTION_ID" ] && [ "$QUESTION_ID" != null ]
curl -fsS "$BASE/exams/$EXAM_ID/questions" -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' --data "$(jq -nc --arg questionId "$QUESTION_ID" '{questionId:$questionId,position:1,points:1}')" | jq -e --arg examId "$EXAM_ID" '.examId==$examId' >/dev/null
curl -fsS "$BASE/exams/$EXAM_ID/publish" -X POST -H "Authorization: Bearer $TOKEN" | jq -e '.status=="published"' >/dev/null
START="$(date -u -d '+5 minutes' +%Y-%m-%dT%H:%M:%SZ)"
END="$(date -u -d '+65 minutes' +%Y-%m-%dT%H:%M:%SZ)"
curl -fsS "$BASE/exams/$EXAM_ID/schedule" -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' --data "$(jq -nc --arg startAt "$START" --arg endAt "$END" '{startAt:$startAt,endAt:$endAt}')" | jq -e '.status=="scheduled"' >/dev/null
printf 'instructor_dashboard_e2e=PASS\nexam_id=%s\nquestion_id=%s\n' "$EXAM_ID" "$QUESTION_ID"
