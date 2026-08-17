#!/usr/bin/env python3
import os
import requests

BASE = os.environ.get("PROCTORING", "http://localhost:8000")
cases = [
    ({"face_count": 0, "camera_available": True, "tab_visible": True, "fullscreen": True, "network_online": True}, "face_missing", "high"),
    ({"face_count": 2, "camera_available": True, "tab_visible": True, "fullscreen": True, "network_online": True}, "multiple_faces", "high"),
    ({"face_count": 1, "camera_available": False, "tab_visible": True, "fullscreen": True, "network_online": True}, "camera_interrupted", "critical"),
    ({"face_count": 1, "camera_available": True, "tab_visible": True, "fullscreen": False, "network_online": True}, "fullscreen_exited", "medium"),
    ({"face_count": 1, "camera_available": True, "tab_visible": False, "fullscreen": True, "network_online": True}, "tab_visibility_changed", "medium"),
    ({"face_count": 1, "camera_available": True, "tab_visible": True, "fullscreen": True, "network_online": False}, "network_offline", "medium"),
    ({"face_count": 1, "camera_available": True, "tab_visible": True, "fullscreen": True, "network_online": True, "network_rtt_ms": 1800}, "network_degraded", "low"),
]
for signal, expected_type, expected_severity in cases:
    signal["attempt_id"] = "matrix-test"
    response = requests.post(f"{BASE}/v1/analyze-signal", json=signal, timeout=10)
    response.raise_for_status()
    result = response.json()
    assert result["event_type"] == expected_type, (expected_type, result)
    assert result["severity"] == expected_severity, (expected_severity, result)
print(f"PROCTORING_SIGNAL_MATRIX_PASS cases={len(cases)}")
