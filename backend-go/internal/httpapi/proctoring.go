package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type proctorClient struct {
	conn      *websocket.Conn
	attemptID string
	send      chan []byte
}

type ProctorHub struct {
	mu      sync.RWMutex
	clients map[*proctorClient]struct{}
	upgrade websocket.Upgrader
}

func NewProctorHub() *ProctorHub {
	return &ProctorHub{
		clients: make(map[*proctorClient]struct{}),
		upgrade: websocket.Upgrader{ReadBufferSize: 2048, WriteBufferSize: 4096, CheckOrigin: func(_ *http.Request) bool { return true }},
	}
}

func (h *ProctorHub) add(c *proctorClient) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}
func (h *ProctorHub) remove(c *proctorClient) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()
	_ = c.conn.Close()
}
func (h *ProctorHub) Broadcast(attemptID string, payload interface{}) {
	data, err := json.Marshal(map[string]interface{}{"type": "proctoring.event", "attemptId": attemptID, "data": payload})
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if c.attemptID != "" && c.attemptID != attemptID {
			continue
		}
		select {
		case c.send <- data:
		default:
		}
	}
}
func (h *ProctorHub) Count() int { h.mu.RLock(); defer h.mu.RUnlock(); return len(h.clients) }

func (s *Server) proctoringWebSocket(w http.ResponseWriter, r *http.Request) {
	if s.Hub == nil {
		http.Error(w, "realtime proctoring unavailable", http.StatusServiceUnavailable)
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" && !s.allowedOrigin(origin) {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return
	}
	attemptFilter := strings.TrimSpace(r.URL.Query().Get("attemptId"))
	if attemptFilter != "" {
		if _, err := uuid.Parse(attemptFilter); err != nil {
			http.Error(w, "invalid attemptId", http.StatusBadRequest)
			return
		}
	}
	conn, err := s.Hub.upgrade.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &proctorClient{conn: conn, attemptID: attemptFilter, send: make(chan []byte, 32)}
	s.Hub.add(client)
	defer s.Hub.remove(client)
	go func() {
		for msg := range client.send {
			_ = conn.SetWriteDeadline(time.Now().Add(8 * time.Second))
			if conn.WriteMessage(websocket.TextMessage, msg) != nil {
				return
			}
		}
	}()
	_ = conn.WriteJSON(map[string]interface{}{"type": "proctoring.ready", "connections": s.Hub.Count()})
	conn.SetReadLimit(4096)
	_ = conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(70 * time.Second)) })
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

type proctorSignalInput struct {
	AttemptID         string                 `json:"attemptId"`
	FaceCount         int                    `json:"faceCount"`
	CameraAvailable   bool                   `json:"cameraAvailable"`
	FacePositionDelta float64                `json:"facePositionDelta"`
	TabVisible        bool                   `json:"tabVisible"`
	Fullscreen        bool                   `json:"fullscreen"`
	NetworkOnline     bool                   `json:"networkOnline"`
	NetworkRTTMs      int                    `json:"networkRttMs"`
	Metadata          map[string]interface{} `json:"metadata"`
}
type proctorReviewInput struct {
	Decision string `json:"decision"`
	Note     string `json:"note"`
}

type proctorEventView struct {
	ID           string                 `json:"id"`
	AttemptID    string                 `json:"attemptId"`
	StudentName  string                 `json:"studentName"`
	ExamTitle    string                 `json:"examTitle"`
	EventType    string                 `json:"eventType"`
	Severity     string                 `json:"severity"`
	RiskScore    float64                `json:"riskScore"`
	Confidence   float64                `json:"confidence"`
	Source       string                 `json:"source"`
	OccurredAt   time.Time              `json:"occurredAt"`
	ReviewStatus string                 `json:"reviewStatus"`
	Metadata     map[string]interface{} `json:"metadata"`
}

func classifyEvent(eventType string, confidence float64, metadata map[string]interface{}) (string, float64) {
	base := map[string]float64{
		"camera_interrupted": 100, "multiple_faces": 78, "face_missing": 72,
		"fullscreen_exited": 52, "tab_visibility_changed": 48, "window_blurred": 34,
		"network_offline": 42, "network_degraded": 18, "face_position_changed": 24,
		"face_detection_error": 20,
	}[strings.ToLower(eventType)]
	if base == 0 {
		base = 10
	}
	risk := base * (0.65 + confidence*0.35)
	if risk > 100 {
		risk = 100
	}
	severity := "info"
	switch {
	case risk >= 85:
		severity = "critical"
	case risk >= 60:
		severity = "high"
	case risk >= 35:
		severity = "medium"
	case risk >= 15:
		severity = "low"
	}
	return severity, risk
}

func (s *Server) recordProctoringEvent(ctx context.Context, attemptID uuid.UUID, input proctorEventInput, source string) (proctorEventView, error) {
	if input.EventType == "" {
		return proctorEventView{}, errors.New("eventType is required")
	}
	if input.Metadata == nil {
		input.Metadata = map[string]interface{}{}
	}
	severity, risk := classifyEvent(input.EventType, input.Confidence, input.Metadata)
	meta, _ := json.Marshal(input.Metadata)
	faceCount := 0
	if value, ok := input.Metadata["faceCount"].(int); ok {
		faceCount = value
	}
	if value, ok := input.Metadata["face_count"].(float64); ok {
		faceCount = int(value)
	}
	var view proctorEventView
	var studentName, examTitle, psID string
	var sessionRisk float64
	if err := s.DB.QueryRowContext(ctx, `SELECT ps.id,u.display_name,e.title,ps.risk_score FROM proctoring_sessions ps JOIN exam_attempts a ON a.id=ps.attempt_id JOIN exam_sessions es ON es.id=a.session_id JOIN users u ON u.id=es.student_id JOIN exams e ON e.id=es.exam_id WHERE a.id=$1`, attemptID).Scan(&psID, &studentName, &examTitle, &sessionRisk); err != nil {
		return view, err
	}
	if risk > sessionRisk {
		sessionRisk = risk
	}
	id := uuid.New()
	_, err := s.DB.ExecContext(ctx, `INSERT INTO proctoring_events(id,proctoring_session_id,event_type,confidence,metadata,severity,risk_score,source,review_status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,'open')`, id, psID, input.EventType, input.Confidence, meta, severity, risk, source)
	if err != nil {
		return view, err
	}
	_, err = s.DB.ExecContext(ctx, `UPDATE proctoring_sessions SET last_signal_at=now(),last_heartbeat_at=now(),connection_status='connected',monitoring_status=CASE WHEN CAST($2 AS numeric) >= 35 THEN 'attention' ELSE 'monitoring' END,risk_score=CAST($2 AS numeric),last_face_count=CAST($3 AS integer) WHERE attempt_id=$1`, attemptID, sessionRisk, faceCount)
	if err != nil {
		if s.Logger != nil {
			s.Logger.Error("update proctoring session failed", "error", err, "attemptId", attemptID, "faceCount", faceCount, "risk", sessionRisk)
		}
		return view, err
	}
	view = proctorEventView{ID: id.String(), AttemptID: attemptID.String(), StudentName: studentName, ExamTitle: examTitle, EventType: input.EventType, Severity: severity, RiskScore: risk, Confidence: input.Confidence, Source: source, OccurredAt: time.Now().UTC(), ReviewStatus: "open", Metadata: input.Metadata}
	if s.Hub != nil {
		s.Hub.Broadcast(attemptID.String(), view)
	}
	return view, nil
}

func (s *Server) proctoringSignal(w http.ResponseWriter, r *http.Request) {
	attemptID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid attempt id"})
		return
	}
	c := claimsOf(r)
	if !s.canAccessAttempt(r.Context(), attemptID, c) {
		write(w, 403, map[string]string{"error": "attempt access denied"})
		return
	}
	var in proctorSignalInput
	if !decode(w, r, &in) {
		return
	}
	if in.FaceCount < 0 || in.FaceCount > 10 {
		write(w, 400, map[string]string{"error": "faceCount must be 0..10"})
		return
	}
	payload, _ := json.Marshal(map[string]interface{}{"attempt_id": attemptID.String(), "face_count": in.FaceCount, "camera_available": in.CameraAvailable, "face_position_delta": in.FacePositionDelta, "tab_visible": in.TabVisible, "fullscreen": in.Fullscreen, "network_online": in.NetworkOnline, "network_rtt_ms": in.NetworkRTTMs, "metadata": in.Metadata})
	url := strings.TrimRight(s.ProctoringURL, "/") + "/v1/analyze-signal"
	request, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, url, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		write(w, 503, map[string]string{"error": "proctoring service unavailable"})
		return
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 256*1024))
	if response.StatusCode >= 300 {
		write(w, 503, map[string]string{"error": "proctoring analysis failed"})
		return
	}
	var result struct {
		Suspicious bool                   `json:"suspicious"`
		EventType  string                 `json:"event_type"`
		Severity   string                 `json:"severity"`
		Confidence float64                `json:"confidence"`
		RiskScore  float64                `json:"risk_score"`
		Evidence   map[string]interface{} `json:"evidence"`
	}
	if json.Unmarshal(body, &result) != nil {
		write(w, 502, map[string]string{"error": "invalid proctoring response"})
		return
	}
	if !result.Suspicious {
		_, _ = s.DB.ExecContext(r.Context(), `UPDATE proctoring_sessions SET last_signal_at=now(),last_heartbeat_at=now(),connection_status='connected',monitoring_status='monitoring',last_face_count=$2 WHERE attempt_id=$1`, attemptID, in.FaceCount)
		write(w, 200, result)
		return
	}
	meta := result.Evidence
	if meta == nil {
		meta = map[string]interface{}{}
	}
	meta["faceCount"] = in.FaceCount
	meta["fullscreen"] = in.Fullscreen
	meta["networkOnline"] = in.NetworkOnline
	event, err := s.recordProctoringEvent(r.Context(), attemptID, proctorEventInput{EventType: result.EventType, Confidence: result.Confidence, Metadata: meta}, "python-signal")
	if err != nil {
		write(w, 500, map[string]string{"error": "could not save analyzed event"})
		return
	}
	write(w, 201, event)
}

func (s *Server) proctoringEvent(w http.ResponseWriter, r *http.Request) {
	attemptID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid attempt id"})
		return
	}
	c := claimsOf(r)
	if !s.canAccessAttempt(r.Context(), attemptID, c) {
		write(w, 403, map[string]string{"error": "attempt access denied"})
		return
	}
	var in proctorEventInput
	if !decode(w, r, &in) {
		return
	}
	if in.EventType == "" || in.Confidence < 0 || in.Confidence > 1 {
		write(w, 400, map[string]string{"error": "eventType and confidence 0..1 are required"})
		return
	}
	event, err := s.recordProctoringEvent(r.Context(), attemptID, in, "browser")
	if err != nil {
		write(w, 500, map[string]string{"error": "could not save proctoring event"})
		return
	}
	s.audit(r.Context(), c.UserID, "PROCTORING_EVENT_CREATED", "proctoring_events", uuid.MustParse(event.ID), "SUCCESS", map[string]interface{}{"attemptId": attemptID.String(), "severity": event.Severity})
	write(w, 201, event)
}

func (s *Server) allowedOrigin(origin string) bool {
	for _, allowed := range strings.Split(s.CORS, ",") {
		if strings.TrimSpace(allowed) == origin {
			return true
		}
	}
	return false
}

func (s *Server) activeProctoring(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(), `SELECT a.id,u.display_name,e.title,a.status,ps.connection_status,ps.monitoring_status,ps.risk_score,ps.last_face_count,COALESCE(COUNT(pe.id) FILTER(WHERE pe.review_status='open'),0),COALESCE(MAX(pe.occurred_at),a.started_at),a.started_at,a.expires_at FROM exam_attempts a JOIN exam_sessions es ON es.id=a.session_id JOIN users u ON u.id=es.student_id JOIN exams e ON e.id=es.exam_id JOIN proctoring_sessions ps ON ps.attempt_id=a.id LEFT JOIN proctoring_events pe ON pe.proctoring_session_id=ps.id WHERE a.status='in_progress' GROUP BY a.id,u.display_name,e.title,a.status,ps.connection_status,ps.monitoring_status,ps.risk_score,ps.last_face_count,ps.last_signal_at,ps.last_heartbeat_at,a.started_at,a.expires_at ORDER BY ps.risk_score DESC,a.started_at ASC`)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not load active monitoring"})
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, name, title, status, connection, monitoring string
		var risk float64
		var faces, open int
		var last, started, expires time.Time
		if rows.Scan(&id, &name, &title, &status, &connection, &monitoring, &risk, &faces, &open, &last, &started, &expires) == nil {
			out = append(out, map[string]interface{}{"attemptId": id, "studentName": name, "examTitle": title, "attemptStatus": status, "connectionStatus": connection, "monitoringStatus": monitoring, "riskScore": risk, "faceCount": faces, "openEvents": open, "lastEventAt": last, "startedAt": started, "expiresAt": expires})
		}
	}
	write(w, 200, out)
}

func (s *Server) proctoringEvents(w http.ResponseWriter, r *http.Request) {
	attemptID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid attempt id"})
		return
	}
	rows, err := s.DB.QueryContext(r.Context(), `SELECT pe.id,a.id,u.display_name,e.title,pe.event_type,pe.severity,pe.risk_score,pe.confidence,pe.source,pe.occurred_at,pe.review_status,pe.metadata FROM proctoring_events pe JOIN proctoring_sessions ps ON ps.id=pe.proctoring_session_id JOIN exam_attempts a ON a.id=ps.attempt_id JOIN exam_sessions es ON es.id=a.session_id JOIN users u ON u.id=es.student_id JOIN exams e ON e.id=es.exam_id WHERE a.id=$1 ORDER BY pe.occurred_at DESC LIMIT 200`, attemptID)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not load monitoring events"})
		return
	}
	defer rows.Close()
	out := []proctorEventView{}
	for rows.Next() {
		var v proctorEventView
		var metadata []byte
		if rows.Scan(&v.ID, &v.AttemptID, &v.StudentName, &v.ExamTitle, &v.EventType, &v.Severity, &v.RiskScore, &v.Confidence, &v.Source, &v.OccurredAt, &v.ReviewStatus, &metadata) == nil {
			_ = json.Unmarshal(metadata, &v.Metadata)
			out = append(out, v)
		}
	}
	write(w, 200, out)
}

func (s *Server) reviewProctoringEvent(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("eventId"))
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid event id"})
		return
	}
	var in proctorReviewInput
	if !decode(w, r, &in) {
		return
	}
	if in.Decision != "confirmed" && in.Decision != "dismissed" && in.Decision != "needs_followup" {
		write(w, 400, map[string]string{"error": "invalid review decision"})
		return
	}
	c := claimsOf(r)
	res, err := s.DB.ExecContext(r.Context(), `UPDATE proctoring_events SET review_status=$1,reviewed_at=now(),reviewed_by=$2,reviewer_note=$3 WHERE id=$4`, in.Decision, c.UserID, in.Note, id)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not review event"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		write(w, 404, map[string]string{"error": "event not found"})
		return
	}
	s.audit(r.Context(), c.UserID, "PROCTORING_EVENT_REVIEWED", "proctoring_events", id, "SUCCESS", map[string]interface{}{"decision": in.Decision})
	write(w, 200, map[string]interface{}{"id": id.String(), "reviewStatus": in.Decision})
}
