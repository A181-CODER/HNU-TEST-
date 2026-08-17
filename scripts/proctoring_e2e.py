#!/usr/bin/env python3
import json
import os
import time
from datetime import datetime, timedelta, timezone

import requests
import websocket

API = os.environ.get("API", "http://localhost:8080/api/v1")
WS = os.environ.get("WS", "ws://localhost:8080/api/v1/proctoring/ws")
PASSWORD = "ChangeMe-Development-Only!"

def login(email):
    response = requests.post(f"{API}/auth/login", json={"email": email, "password": PASSWORD}, timeout=10)
    response.raise_for_status()
    return response.json()["accessToken"]

def call(method, path, token, **kwargs):
    response = requests.request(method, f"{API}{path}", headers={"Authorization": f"Bearer {token}"}, timeout=15, **kwargs)
    response.raise_for_status()
    return response.json() if response.content else {}

def main():
    instructor = login("instructor@hnu-test.local")
    student = login("student@hnu-test.local")
    proctor = login("proctor@hnu-test.local")
    ws = websocket.create_connection(f"{WS}?token={proctor}", origin="http://localhost:5173", timeout=8)
    ready = json.loads(ws.recv())
    assert ready["type"] == "proctoring.ready", ready

    exam = call("POST", "/exams", instructor, json={"title": "Phase 3 Proctoring E2E", "courseCode": "SWE-201", "durationMinutes": 30, "attemptLimit": 1, "passingScore": 60, "totalMarks": 2, "negativeMarking": 0.5, "randomizeQuestions": True, "randomizeOptions": True, "allowReview": True, "resultVisibility": "not_published", "instructions": "Phase 3 acceptance"})
    exam_id = exam["id"]
    call("POST", f"/exams/{exam_id}/questions", instructor, json={"questionId": "60000000-0000-0000-0000-000000000001", "position": 1, "points": 2})
    call("POST", f"/exams/{exam_id}/publish", instructor)
    now = datetime.now(timezone.utc)
    call("POST", f"/exams/{exam_id}/schedule", instructor, json={"startAt": (now - timedelta(minutes=1)).isoformat(), "endAt": (now + timedelta(minutes=30)).isoformat()})
    attempt = call("POST", f"/exams/{exam_id}/start", student)
    attempt_id = attempt["id"]

    signal = call("POST", f"/attempts/{attempt_id}/proctoring-signal", student, json={"attemptId": attempt_id, "faceCount": 2, "cameraAvailable": True, "facePositionDelta": 0, "tabVisible": True, "fullscreen": True, "networkOnline": True, "networkRttMs": 42, "metadata": {"test": "multiple-face"}})
    assert signal["eventType"] == "multiple_faces", signal
    assert signal["severity"] == "high", signal
    assert signal["riskScore"] >= 60, signal

    event_message = None
    deadline = time.time() + 8
    while time.time() < deadline:
        message = json.loads(ws.recv())
        if message.get("type") == "proctoring.event" and message.get("attemptId") == attempt_id:
            event_message = message["data"]
            break
    assert event_message, "WebSocket did not broadcast analyzed event"
    assert event_message["severity"] == "high", event_message

    active = call("GET", "/proctoring/active-attempts", proctor)
    live = next(item for item in active if item["attemptId"] == attempt_id)
    assert live["riskScore"] >= 60 and live["openEvents"] >= 1, live

    events = call("GET", f"/proctoring/attempts/{attempt_id}/events", proctor)
    assert any(item["id"] == event_message["id"] for item in events), events
    reviewed = call("POST", f"/proctoring/events/{event_message['id']}/review", proctor, json={"decision": "confirmed", "note": "E2E human review"})
    assert reviewed["reviewStatus"] == "confirmed", reviewed

    answer = call("PATCH", f"/attempts/{attempt_id}/answers/60000000-0000-0000-0000-000000000001", student, json={"questionId": "60000000-0000-0000-0000-000000000001", "values": ["A"], "markedForReview": True})
    assert answer["status"] == "saved", answer
    result = call("POST", f"/attempts/{attempt_id}/submit", student)
    assert result["grade"] == "A" and result["correct"] == 1, result
    ws.close()
    print(json.dumps({"status": "PROCTORING_E2E_PASS", "attemptId": attempt_id, "eventId": event_message["id"], "severity": event_message["severity"], "riskScore": event_message["riskScore"], "resultGrade": result["grade"]}))

if __name__ == "__main__":
    main()
