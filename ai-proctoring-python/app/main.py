from datetime import datetime, timezone
from enum import Enum
from typing import Any
from fastapi import FastAPI
from pydantic import BaseModel, Field

app = FastAPI(title="HNU TEST Proctoring Event Service", version="0.1.0")
class EventType(str, Enum):
    face_missing="face_missing"; multiple_faces="multiple_faces"; camera_interrupted="camera_interrupted"; tab_visibility_changed="tab_visibility_changed"; window_blurred="window_blurred"; face_position_changed="face_position_changed"
class FrameSignal(BaseModel):
    attempt_id: str
    face_count: int = Field(ge=0, le=10)
    camera_available: bool = True
    face_position_delta: float = Field(default=0, ge=0)
    tab_visible: bool = True
    client_timestamp: datetime | None = None
    metadata: dict[str, Any] = {}
class SuspiciousEvent(BaseModel):
    suspicious: bool
    event_type: EventType | None = None
    confidence: float = Field(ge=0, le=1)
    timestamp: datetime
    evidence: dict[str, Any]

@app.get("/health")
def health(): return {"status":"ok","service":"python-proctoring"}
@app.post("/v1/analyze-signal", response_model=SuspiciousEvent)
def analyze_signal(signal: FrameSignal):
    event = None; confidence = 0.0; evidence = {"face_count":signal.face_count,"camera_available":signal.camera_available,"tab_visible":signal.tab_visible}
    if not signal.camera_available: event, confidence = EventType.camera_interrupted, 1.0
    elif signal.face_count == 0: event, confidence = EventType.face_missing, 0.82
    elif signal.face_count > 1: event, confidence = EventType.multiple_faces, min(0.99, 0.75 + signal.face_count*0.05)
    elif not signal.tab_visible: event, confidence = EventType.tab_visibility_changed, 0.96
    elif signal.face_position_delta > 0.35: event, confidence = EventType.face_position_changed, min(0.95, signal.face_position_delta)
    return SuspiciousEvent(suspicious=event is not None,event_type=event,confidence=confidence,timestamp=datetime.now(timezone.utc),evidence=evidence)
