from datetime import datetime, timezone
from enum import Enum
from typing import Any

from fastapi import FastAPI
from pydantic import BaseModel, Field

app = FastAPI(title="HNU TEST Proctoring Event Service", version="0.2.0")


class EventType(str, Enum):
    face_missing = "face_missing"
    multiple_faces = "multiple_faces"
    camera_interrupted = "camera_interrupted"
    tab_visibility_changed = "tab_visibility_changed"
    window_blurred = "window_blurred"
    fullscreen_exited = "fullscreen_exited"
    network_offline = "network_offline"
    network_degraded = "network_degraded"
    face_position_changed = "face_position_changed"
    face_detection_error = "face_detection_error"


class Severity(str, Enum):
    info = "info"
    low = "low"
    medium = "medium"
    high = "high"
    critical = "critical"


class FrameSignal(BaseModel):
    attempt_id: str
    face_count: int = Field(ge=0, le=10)
    camera_available: bool = True
    face_position_delta: float = Field(default=0, ge=0)
    tab_visible: bool = True
    fullscreen: bool = True
    network_online: bool = True
    network_rtt_ms: int | None = Field(default=None, ge=0, le=120000)
    client_timestamp: datetime | None = None
    metadata: dict[str, Any] = Field(default_factory=dict)


class SuspiciousEvent(BaseModel):
    suspicious: bool
    event_type: EventType | None = None
    severity: Severity
    confidence: float = Field(ge=0, le=1)
    risk_score: float = Field(ge=0, le=100)
    timestamp: datetime
    evidence: dict[str, Any]


@app.get("/health")
def health():
    return {"status": "ok", "service": "python-proctoring", "version": "0.2.0"}


def candidate(signal: FrameSignal) -> tuple[EventType | None, Severity, float, float]:
    if not signal.camera_available:
        return EventType.camera_interrupted, Severity.critical, 1.0, 100.0
    if signal.face_count > 1:
        return EventType.multiple_faces, Severity.high, min(0.99, 0.75 + signal.face_count * 0.05), min(100.0, 72.0 + signal.face_count * 7.0)
    if signal.face_count == 0:
        return EventType.face_missing, Severity.high, 0.88, 78.0
    if not signal.fullscreen:
        return EventType.fullscreen_exited, Severity.medium, 0.98, 52.0
    if not signal.tab_visible:
        return EventType.tab_visibility_changed, Severity.medium, 0.96, 48.0
    if not signal.network_online:
        return EventType.network_offline, Severity.medium, 1.0, 42.0
    if signal.network_rtt_ms is not None and signal.network_rtt_ms > 1500:
        return EventType.network_degraded, Severity.low, 0.9, 18.0
    if signal.face_position_delta > 0.35:
        return EventType.face_position_changed, Severity.low, min(0.95, signal.face_position_delta), min(35.0, signal.face_position_delta * 70)
    return None, Severity.info, 0.0, 0.0


@app.post("/v1/analyze-signal", response_model=SuspiciousEvent)
def analyze_signal(signal: FrameSignal):
    event, severity, confidence, risk_score = candidate(signal)
    evidence = {
        "face_count": signal.face_count,
        "camera_available": signal.camera_available,
        "tab_visible": signal.tab_visible,
        "fullscreen": signal.fullscreen,
        "network_online": signal.network_online,
        "network_rtt_ms": signal.network_rtt_ms,
        "metadata": signal.metadata,
    }
    return SuspiciousEvent(
        suspicious=event is not None,
        event_type=event,
        severity=severity,
        confidence=confidence,
        risk_score=risk_score,
        timestamp=datetime.now(timezone.utc),
        evidence=evidence,
    )
