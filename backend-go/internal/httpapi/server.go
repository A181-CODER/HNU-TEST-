package httpapi

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/A181-CODER/HNU-TEST-/backend-go/internal/auth"
	"github.com/A181-CODER/HNU-TEST-/backend-go/internal/grading"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Server struct {
	DB            *sql.DB
	Auth          auth.Service
	Logger        *slog.Logger
	CORS          string
	Hub           *ProctorHub
	ProctoringURL string
}

type contextKey string

const claimsKey contextKey = "claims"

type sqlExecer interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type loginResponse struct {
	AccessToken string       `json:"accessToken"`
	User        userResponse `json:"user"`
}
type userResponse struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}
type option struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}
type examQuestion struct {
	ID       string   `json:"id"`
	Position int      `json:"position"`
	Type     string   `json:"type"`
	Prompt   string   `json:"prompt"`
	Points   float64  `json:"points"`
	Options  []option `json:"options,omitempty"`
}
type answerInput struct {
	QuestionID      string   `json:"questionId"`
	Values          []string `json:"values"`
	TextAnswer      string   `json:"textAnswer"`
	MarkedForReview bool     `json:"markedForReview"`
}
type attemptResponse struct {
	ID          string                 `json:"id"`
	ExamID      string                 `json:"examId"`
	ExamTitle   string                 `json:"examTitle"`
	Status      string                 `json:"status"`
	StartedAt   time.Time              `json:"startedAt"`
	ExpiresAt   time.Time              `json:"expiresAt"`
	ServerTime  time.Time              `json:"serverTime"`
	AllowReview bool                   `json:"allowReview"`
	Questions   []examQuestion         `json:"questions"`
	Answers     map[string]answerInput `json:"answers"`
}
type resultResponse struct {
	ID              string    `json:"id"`
	AttemptID       string    `json:"attemptId"`
	ExamTitle       string    `json:"examTitle"`
	Score           float64   `json:"score"`
	Maximum         float64   `json:"maximum"`
	Percentage      float64   `json:"percentage"`
	Grade           string    `json:"grade"`
	Correct         int       `json:"correct"`
	Incorrect       int       `json:"incorrect"`
	Unanswered      int       `json:"unanswered"`
	SubmittedAt     time.Time `json:"submittedAt"`
	DurationSeconds int       `json:"durationSeconds"`
	Published       bool      `json:"published"`
}

type createExamRequest struct {
	CourseID           string     `json:"courseId"`
	Title              string     `json:"title"`
	Description        string     `json:"description"`
	CourseCode         string     `json:"courseCode"`
	DurationMinutes    int        `json:"durationMinutes"`
	StartAt            *time.Time `json:"startAt"`
	EndAt              *time.Time `json:"endAt"`
	AttemptLimit       int        `json:"attemptLimit"`
	PassingScore       float64    `json:"passingScore"`
	TotalMarks         float64    `json:"totalMarks"`
	NegativeMarking    float64    `json:"negativeMarking"`
	RandomizeQuestions bool       `json:"randomizeQuestions"`
	RandomizeOptions   bool       `json:"randomizeOptions"`
	AllowReview        bool       `json:"allowReview"`
	ResultVisibility   string     `json:"resultVisibility"`
	Instructions       string     `json:"instructions"`
}
type scheduleRequest struct {
	StartAt time.Time `json:"startAt"`
	EndAt   time.Time `json:"endAt"`
}
type questionInput struct {
	BankID      string   `json:"bankId"`
	Type        string   `json:"type"`
	Prompt      string   `json:"prompt"`
	Explanation string   `json:"explanation"`
	Difficulty  string   `json:"difficulty"`
	Points      float64  `json:"points"`
	Tags        []string `json:"tags"`
	Options     []struct {
		Key       string `json:"key"`
		Text      string `json:"text"`
		IsCorrect bool   `json:"isCorrect"`
	} `json:"options"`
}

type proctorEventInput struct {
	EventType  string                 `json:"eventType"`
	Confidence float64                `json:"confidence"`
	Metadata   map[string]interface{} `json:"metadata"`
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /ready", s.ready)
	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.Handle("GET /api/v1/me", s.requireRole(http.HandlerFunc(s.me)))
	mux.Handle("GET /api/v1/organization/tree", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor", "proctor", "student"})(http.HandlerFunc(s.organizationTree)))
	mux.Handle("GET /api/v1/organization/overview", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor", "proctor"})(http.HandlerFunc(s.organizationOverview)))
	mux.Handle("POST /api/v1/organization/assignments", s.requireAnyRole([]string{"super_admin", "university_admin"})(http.HandlerFunc(s.organizationAssignments)))
	mux.Handle("POST /api/v1/courses/{id}/students", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor"})(http.HandlerFunc(s.enrollCourseStudent)))
	mux.Handle("POST /api/v1/courses/{id}/instructors", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor"})(http.HandlerFunc(s.assignCourseInstructor)))
	mux.Handle("POST /api/v1/exams/{id}/proctors", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor"})(http.HandlerFunc(s.assignExamProctor)))
	mux.Handle("GET /api/v1/exams", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor"})(http.HandlerFunc(s.exams)))
	mux.Handle("POST /api/v1/exams", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor"})(http.HandlerFunc(s.createExam)))
	mux.Handle("GET /api/v1/exams/{id}", s.requireRole(http.HandlerFunc(s.examDetails)))
	mux.Handle("PATCH /api/v1/exams/{id}", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor"})(http.HandlerFunc(s.updateExam)))
	mux.Handle("POST /api/v1/exams/{id}/publish", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor"})(http.HandlerFunc(s.publishExam)))
	mux.Handle("POST /api/v1/exams/{id}/schedule", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor"})(http.HandlerFunc(s.scheduleExam)))
	mux.Handle("POST /api/v1/exams/{id}/questions", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor"})(http.HandlerFunc(s.attachQuestion)))
	mux.Handle("GET /api/v1/questions", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor"})(http.HandlerFunc(s.questions)))
	mux.Handle("POST /api/v1/questions", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor"})(http.HandlerFunc(s.createQuestion)))
	mux.Handle("GET /api/v1/student/exams", s.requireAnyRole([]string{"student"})(http.HandlerFunc(s.studentExams)))
	mux.Handle("POST /api/v1/exams/{id}/start", s.requireAnyRole([]string{"student"})(http.HandlerFunc(s.startExam)))
	mux.Handle("GET /api/v1/attempts/{id}", s.requireRole(http.HandlerFunc(s.getAttempt)))
	mux.Handle("POST /api/v1/attempts/{id}/answers", s.requireRole(http.HandlerFunc(s.saveAnswer)))
	mux.Handle("PATCH /api/v1/attempts/{id}/answers/{questionId}", s.requireRole(http.HandlerFunc(s.saveAnswer)))
	mux.Handle("POST /api/v1/attempts/{id}/submit", s.requireRole(http.HandlerFunc(s.submitAttempt)))
	mux.Handle("GET /api/v1/attempts/{id}/result", s.requireRole(http.HandlerFunc(s.attemptResult)))
	mux.Handle("POST /api/v1/attempts/{id}/proctoring-events", s.requireRole(http.HandlerFunc(s.proctoringEvent)))
	mux.Handle("POST /api/v1/attempts/{id}/proctoring-signal", s.requireRole(http.HandlerFunc(s.proctoringSignal)))
	mux.Handle("GET /api/v1/proctoring/active-attempts", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor", "proctor"})(http.HandlerFunc(s.activeProctoring)))
	mux.Handle("GET /api/v1/proctoring/attempts/{id}/events", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor", "proctor"})(http.HandlerFunc(s.proctoringEvents)))
	mux.Handle("POST /api/v1/proctoring/events/{eventId}/review", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor", "proctor"})(http.HandlerFunc(s.reviewProctoringEvent)))
	mux.Handle("GET /api/v1/instructor/exams/{id}/results", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor", "proctor"})(http.HandlerFunc(s.instructorResults)))
	mux.Handle("GET /api/v1/proctoring/ws", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor", "proctor"})(http.HandlerFunc(s.proctoringWebSocket)))
	mux.Handle("POST /api/v1/results/{id}/publish", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor"})(http.HandlerFunc(s.publishResult)))
	return s.middleware(mux)
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self' http://localhost:8080 http://localhost:8000 ws://localhost:8080; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'")
		w.Header().Set("Access-Control-Allow-Origin", s.CORS)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method == http.MethodGet {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	write(w, http.StatusOK, map[string]string{"status": "ok", "service": "hnu-api"})
}
func (s *Server) ready(w http.ResponseWriter, _ *http.Request) {
	if s.DB != nil {
		if err := s.DB.Ping(); err != nil {
			write(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
			return
		}
	}
	write(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in loginRequest
	if json.NewDecoder(r.Body).Decode(&in) != nil || !strings.Contains(in.Email, "@") {
		write(w, http.StatusBadRequest, map[string]string{"error": "invalid credentials format"})
		return
	}
	if s.DB == nil {
		write(w, http.StatusServiceUnavailable, map[string]string{"error": "database is not configured"})
		return
	}
	var id, name, email, hash string
	var rolesJSON []byte
	err := s.DB.QueryRowContext(r.Context(), `SELECT u.id,u.display_name,u.email,u.password_hash,COALESCE(json_agg(r.slug) FILTER (WHERE r.slug IS NOT NULL),'[]') FROM users u LEFT JOIN user_roles ur ON ur.user_id=u.id LEFT JOIN roles r ON r.id=ur.role_id WHERE lower(u.email)=lower($1) AND u.deleted_at IS NULL AND u.is_active GROUP BY u.id`, in.Email).Scan(&id, &name, &email, &hash, &rolesJSON)
	if err != nil || !auth.CheckPassword(hash, in.Password) {
		write(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}
	var roles []string
	_ = json.Unmarshal(rolesJSON, &roles)
	token, err := s.Auth.Issue(id, roles)
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"error": "could not issue session"})
		return
	}
	_, _ = s.DB.ExecContext(r.Context(), `INSERT INTO audit_logs(id,user_id,action,result,metadata) VALUES($1,$2,'LOGIN','SUCCESS','{}')`, uuid.New(), id)
	write(w, http.StatusOK, loginResponse{AccessToken: token, User: userResponse{ID: id, Name: name, Email: email, Roles: roles}})
}
func (s *Server) requireRole(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := s.claims(r)
		if err != nil {
			write(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey, c)))
	})
}
func (s *Server) requireAnyRole(roles []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := s.claims(r)
			if err != nil {
				write(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
				return
			}
			if !auth.HasRole(c.Roles, roles...) {
				write(w, http.StatusForbidden, map[string]string{"error": "insufficient permissions"})
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey, c)))
		})
	}
}
func (s *Server) claims(r *http.Request) (*auth.Claims, error) {
	authorization := r.Header.Get("Authorization")
	if authorization == "" && r.URL.Path == "/api/v1/proctoring/ws" {
		if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
			authorization = "Bearer " + token
		}
	}
	return s.Auth.Parse(authorization)
}
func claimsOf(r *http.Request) *auth.Claims {
	c, _ := r.Context().Value(claimsKey).(*auth.Claims)
	return c
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	c := claimsOf(r)
	write(w, http.StatusOK, map[string]interface{}{"id": c.UserID, "roles": c.Roles})
}
func (s *Server) exams(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		write(w, http.StatusOK, []interface{}{})
		return
	}
	c := claimsOf(r)
	rows, err := s.DB.QueryContext(r.Context(), `SELECT e.id,e.title,e.course_code,e.status,e.start_at,e.end_at,e.duration_minutes,e.attempt_limit,e.passing_score,e.total_marks,e.randomize_questions,e.randomize_options FROM exams e LEFT JOIN courses co ON co.id=e.course_id LEFT JOIN departments d ON d.id=co.department_id LEFT JOIN faculties f ON f.id=d.faculty_id WHERE e.deleted_at IS NULL AND ($2 OR EXISTS(SELECT 1 FROM resource_memberships rm WHERE rm.user_id=$1 AND (rm.course_id=co.id OR rm.department_id=d.id OR rm.faculty_id=f.id OR rm.university_id=f.university_id))) ORDER BY e.start_at NULLS LAST,e.created_at DESC`, c.UserID, hasRole(c, "super_admin"))
	if err != nil {
		write(w, 500, map[string]string{"error": "could not load exams"})
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, title, course, status string
		var start, end sql.NullTime
		var dur, limit int
		var passing, total float64
		var rq, ro bool
		if rows.Scan(&id, &title, &course, &status, &start, &end, &dur, &limit, &passing, &total, &rq, &ro) == nil {
			out = append(out, map[string]interface{}{"id": id, "title": title, "courseCode": course, "status": status, "startAt": timeValue(start), "endAt": timeValue(end), "durationMinutes": dur, "attemptLimit": limit, "passingScore": passing, "totalMarks": total, "randomizeQuestions": rq, "randomizeOptions": ro})
		}
	}
	write(w, 200, out)
}
func (s *Server) createExam(w http.ResponseWriter, r *http.Request) {
	var in createExamRequest
	if !decode(w, r, &in) {
		return
	}
	if err := validateExam(in); err != nil {
		write(w, 400, map[string]string{"error": err.Error()})
		return
	}
	c := claimsOf(r)
	if s.DB == nil {
		write(w, 503, map[string]string{"error": "database is not configured"})
		return
	}
	var courseID uuid.UUID
	var err error
	if strings.TrimSpace(in.CourseID) != "" {
		courseID, err = uuid.Parse(in.CourseID)
	} else {
		err = s.DB.QueryRowContext(r.Context(), `SELECT id FROM courses WHERE code=$1`, in.CourseCode).Scan(&courseID)
	}
	if err != nil || courseID == uuid.Nil {
		write(w, 400, map[string]string{"error": "a valid courseId or courseCode is required"})
		return
	}
	if !s.canManageCourse(r.Context(), c, courseID) {
		write(w, 403, map[string]string{"error": "course scope denied"})
		return
	}
	id := uuid.New()
	limit := in.AttemptLimit
	if limit == 0 {
		limit = 1
	}
	visibility := in.ResultVisibility
	if visibility == "" {
		visibility = "not_published"
	}
	_, err = s.DB.ExecContext(r.Context(), `INSERT INTO exams(id,created_by,course_id,title,description,course_code,duration_minutes,start_at,end_at,attempt_limit,passing_score,total_marks,negative_marking,randomize_questions,randomize_options,allow_review,result_visibility,instructions,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,'draft')`, id, c.UserID, courseID, in.Title, in.Description, in.CourseCode, in.DurationMinutes, in.StartAt, in.EndAt, limit, in.PassingScore, in.TotalMarks, in.NegativeMarking, in.RandomizeQuestions, in.RandomizeOptions, in.AllowReview, visibility, in.Instructions)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not create exam"})
		return
	}
	s.audit(r.Context(), c.UserID, "EXAM_CREATED", "exams", id, "SUCCESS", map[string]interface{}{})
	write(w, 201, map[string]interface{}{"id": id, "status": "draft"})
}
func (s *Server) updateExam(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid exam id"})
		return
	}
	var in createExamRequest
	if !decode(w, r, &in) {
		return
	}
	if err := validateExam(in); err != nil {
		write(w, 400, map[string]string{"error": err.Error()})
		return
	}
	c := claimsOf(r)
	if s.DB == nil {
		write(w, 503, map[string]string{"error": "database is not configured"})
		return
	}
	if !s.canManageExam(r.Context(), c, id) {
		write(w, 403, map[string]string{"error": "exam scope denied"})
		return
	}
	res, err := s.DB.ExecContext(r.Context(), `UPDATE exams SET title=$1,description=$2,course_code=$3,duration_minutes=$4,start_at=$5,end_at=$6,attempt_limit=$7,passing_score=$8,total_marks=$9,negative_marking=$10,randomize_questions=$11,randomize_options=$12,allow_review=$13,result_visibility=$14,instructions=$15,updated_at=now() WHERE id=$16 AND status='draft' AND deleted_at IS NULL`, in.Title, in.Description, in.CourseCode, in.DurationMinutes, in.StartAt, in.EndAt, max(in.AttemptLimit, 1), in.PassingScore, in.TotalMarks, in.NegativeMarking, in.RandomizeQuestions, in.RandomizeOptions, in.AllowReview, defaultVisibility(in.ResultVisibility), in.Instructions, id)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not update exam"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		write(w, 409, map[string]string{"error": "only your draft exams can be modified"})
		return
	}
	write(w, 200, map[string]string{"id": id.String(), "status": "draft"})
}
func (s *Server) publishExam(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid exam id"})
		return
	}
	c := claimsOf(r)
	if !s.canManageExam(r.Context(), c, id) {
		write(w, 403, map[string]string{"error": "exam scope denied"})
		return
	}
	res, err := s.DB.ExecContext(r.Context(), `UPDATE exams SET status='published',published_at=now(),updated_at=now() WHERE id=$1 AND status='draft' AND EXISTS(SELECT 1 FROM exam_questions WHERE exam_id=$1)`, id)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not publish exam"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		write(w, 409, map[string]string{"error": "exam must be a draft with at least one question"})
		return
	}
	s.audit(r.Context(), c.UserID, "EXAM_PUBLISHED", "exams", id, "SUCCESS", map[string]interface{}{})
	write(w, 200, map[string]string{"id": id.String(), "status": "published"})
}
func (s *Server) scheduleExam(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid exam id"})
		return
	}
	var in scheduleRequest
	if !decode(w, r, &in) {
		return
	}
	if in.EndAt.Before(in.StartAt) || in.StartAt.Before(time.Now().Add(-2*time.Minute)) {
		write(w, 400, map[string]string{"error": "invalid schedule window"})
		return
	}
	c := claimsOf(r)
	if !s.canManageExam(r.Context(), c, id) {
		write(w, 403, map[string]string{"error": "exam scope denied"})
		return
	}
	res, err := s.DB.ExecContext(r.Context(), `UPDATE exams SET status='scheduled',start_at=$1,end_at=$2,updated_at=now() WHERE id=$3 AND status IN ('published','scheduled')`, in.StartAt, in.EndAt, id)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not schedule exam"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		write(w, 409, map[string]string{"error": "exam is not publishable or not owned by instructor"})
		return
	}
	s.audit(r.Context(), c.UserID, "EXAM_SCHEDULED", "exams", id, "SUCCESS", map[string]interface{}{})
	write(w, 200, map[string]string{"id": id.String(), "status": "scheduled"})
}

func (s *Server) examDetails(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid exam id"})
		return
	}
	if !s.canAccessExam(r.Context(), claimsOf(r), id) {
		write(w, 403, map[string]string{"error": "exam scope denied"})
		return
	}
	var title, course, status, instructions string
	var dur, limit int
	var start, end sql.NullTime
	var passing, total, negative float64
	var rq, ro, review bool
	err = s.DB.QueryRowContext(r.Context(), `SELECT title,course_code,status,instructions,duration_minutes,attempt_limit,start_at,end_at,passing_score,total_marks,negative_marking,randomize_questions,randomize_options,allow_review FROM exams WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&title, &course, &status, &instructions, &dur, &limit, &start, &end, &passing, &total, &negative, &rq, &ro, &review)
	if err != nil {
		write(w, 404, map[string]string{"error": "exam not found"})
		return
	}
	write(w, 200, map[string]interface{}{"id": id, "title": title, "courseCode": course, "status": status, "instructions": instructions, "durationMinutes": dur, "attemptLimit": limit, "startAt": timeValue(start), "endAt": timeValue(end), "passingScore": passing, "totalMarks": total, "negativeMarking": negative, "randomizeQuestions": rq, "randomizeOptions": ro, "allowReview": review})
}

func (s *Server) questions(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(), `SELECT q.id,q.type,q.prompt,q.difficulty,q.points,COALESCE(q.tags,'[]') FROM questions q WHERE q.deleted_at IS NULL ORDER BY q.created_at DESC`)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not load questions"})
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, typ, prompt, diff string
		var points float64
		var tags []byte
		if rows.Scan(&id, &typ, &prompt, &diff, &points, &tags) == nil {
			var tv interface{}
			_ = json.Unmarshal(tags, &tv)
			out = append(out, map[string]interface{}{"id": id, "type": typ, "prompt": prompt, "difficulty": diff, "points": points, "tags": tv})
		}
	}
	write(w, 200, out)
}
func (s *Server) createQuestion(w http.ResponseWriter, r *http.Request) {
	var in questionInput
	if !decode(w, r, &in) {
		return
	}
	if !validQuestionType(in.Type) || strings.TrimSpace(in.Prompt) == "" || in.Points <= 0 {
		write(w, 400, map[string]string{"error": "valid type, prompt and positive points are required"})
		return
	}
	c := claimsOf(r)
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not begin question transaction"})
		return
	}
	defer tx.Rollback()
	id := uuid.New()
	tags, _ := json.Marshal(in.Tags)
	_, err = tx.ExecContext(r.Context(), `INSERT INTO questions(id,bank_id,author_id,type,prompt,explanation,difficulty,points,tags) VALUES($1,NULLIF($2,'')::uuid,$3,$4,$5,$6,COALESCE(NULLIF($7,''),'medium'),$8,$9)`, id, in.BankID, c.UserID, in.Type, in.Prompt, in.Explanation, in.Difficulty, in.Points, tags)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not create question"})
		return
	}
	for _, o := range in.Options {
		if strings.TrimSpace(o.Key) == "" || strings.TrimSpace(o.Text) == "" {
			continue
		}
		if _, err = tx.ExecContext(r.Context(), `INSERT INTO question_options(question_id,option_key,option_text,is_correct) VALUES($1,$2,$3,$4)`, id, o.Key, o.Text, o.IsCorrect); err != nil {
			write(w, 500, map[string]string{"error": "could not create question options"})
			return
		}
	}
	if err = tx.Commit(); err != nil {
		write(w, 500, map[string]string{"error": "could not commit question"})
		return
	}
	s.audit(r.Context(), c.UserID, "QUESTION_CREATED", "questions", id, "SUCCESS", map[string]interface{}{})
	write(w, 201, map[string]string{"id": id.String()})
}
func (s *Server) attachQuestion(w http.ResponseWriter, r *http.Request) {
	examID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid exam id"})
		return
	}
	var in struct {
		QuestionID string  `json:"questionId"`
		Position   int     `json:"position"`
		Points     float64 `json:"points"`
	}
	if !decode(w, r, &in) {
		return
	}
	qid, err := parseUUID(in.QuestionID)
	if err != nil || in.Points <= 0 {
		write(w, 400, map[string]string{"error": "valid questionId and points are required"})
		return
	}
	c := claimsOf(r)
	if !s.canManageExam(r.Context(), c, examID) {
		write(w, 403, map[string]string{"error": "exam scope denied"})
		return
	}
	if in.Position < 1 {
		in.Position = 1
	}
	_, err = s.DB.ExecContext(r.Context(), `INSERT INTO exam_questions(exam_id,question_id,position,points) SELECT $1,$2,$3,$4 WHERE EXISTS(SELECT 1 FROM exams WHERE id=$1 AND status='draft') ON CONFLICT(exam_id,question_id) DO UPDATE SET position=EXCLUDED.position,points=EXCLUDED.points`, examID, qid, in.Position, in.Points)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not attach question"})
		return
	}
	write(w, 201, map[string]string{"examId": examID.String(), "questionId": qid.String()})
}

func (s *Server) studentExams(w http.ResponseWriter, r *http.Request) {
	c := claimsOf(r)
	rows, err := s.DB.QueryContext(r.Context(), `SELECT e.id,e.title,e.course_code,e.status,e.start_at,e.end_at,e.duration_minutes,e.attempt_limit,COUNT(a.id),COUNT(a.id) FILTER(WHERE a.status IN ('submitted','auto_submitted','expired')) FROM exams e JOIN course_students cs ON cs.course_id=e.course_id AND cs.student_id=$1 LEFT JOIN exam_sessions es ON es.exam_id=e.id AND es.student_id=$1 LEFT JOIN exam_attempts a ON a.session_id=es.id WHERE e.deleted_at IS NULL AND e.status IN ('published','scheduled','active','ended') GROUP BY e.id ORDER BY e.start_at NULLS LAST`, c.UserID)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not load student exams"})
		return
	}
	defer rows.Close()
	now := time.Now()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, title, course, status string
		var start, end sql.NullTime
		var dur, limit, attempts, completed int
		if rows.Scan(&id, &title, &course, &status, &start, &end, &dur, &limit, &attempts, &completed) != nil {
			continue
		}
		state := "upcoming"
		if completed > 0 {
			state = "completed"
		} else if start.Valid && now.Before(start.Time) {
			state = "upcoming"
		} else if end.Valid && now.After(end.Time) {
			state = "expired"
		} else {
			state = "available"
		}
		out = append(out, map[string]interface{}{"id": id, "title": title, "courseCode": course, "status": state, "examStatus": status, "startAt": timeValue(start), "endAt": timeValue(end), "durationMinutes": dur, "attemptLimit": limit, "attempts": attempts, "remainingAttempts": max(limit-attempts, 0)})
	}
	write(w, 200, out)
}

func (s *Server) startExam(w http.ResponseWriter, r *http.Request) {
	examID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid exam id"})
		return
	}
	student := claimsOf(r)
	if !s.canAccessExam(r.Context(), student, examID) {
		write(w, 403, map[string]string{"error": "student is not enrolled in this course"})
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		write(w, 500, map[string]string{"error": "could not start transaction"})
		return
	}
	defer tx.Rollback()
	now := time.Now()
	var title, status string
	var duration, limit int
	var start, end sql.NullTime
	var rq, ro, allowReview bool
	err = tx.QueryRowContext(r.Context(), `SELECT title,status,duration_minutes,attempt_limit,start_at,end_at,randomize_questions,randomize_options,allow_review FROM exams WHERE id=$1 AND deleted_at IS NULL FOR SHARE`, examID).Scan(&title, &status, &duration, &limit, &start, &end, &rq, &ro, &allowReview)
	if err != nil {
		write(w, 404, map[string]string{"error": "exam not found"})
		return
	}
	if status != "published" && status != "scheduled" && status != "active" {
		write(w, 409, map[string]string{"error": "exam is not available"})
		return
	}
	if start.Valid && now.Before(start.Time) {
		write(w, 409, map[string]string{"error": "exam has not started"})
		return
	}
	if end.Valid && now.After(end.Time) {
		write(w, 409, map[string]string{"error": "exam schedule has ended"})
		return
	}
	var existingID, existingStatus, title2 string
	var existingDeadline sql.NullTime
	err = tx.QueryRowContext(r.Context(), `SELECT a.id,a.status,a.expires_at,e.title FROM exam_attempts a JOIN exam_sessions es ON es.id=a.session_id JOIN exams e ON e.id=es.exam_id WHERE es.exam_id=$1 AND es.student_id=$2 ORDER BY a.attempt_number DESC LIMIT 1`, examID, student.UserID).Scan(&existingID, &existingStatus, &existingDeadline, &title2)
	if err == nil && existingStatus == "in_progress" && existingDeadline.Valid && now.Before(existingDeadline.Time) {
		existingUUID, parseErr := parseUUID(existingID)
		if parseErr != nil {
			write(w, http.StatusInternalServerError, map[string]string{"error": "invalid stored attempt id"})
			return
		}
		resp, err := s.attemptPayloadTx(r.Context(), tx, existingUUID, now)
		if err != nil {
			write(w, 500, map[string]string{"error": "could not resume attempt"})
			return
		}
		if err = tx.Commit(); err != nil {
			write(w, 500, map[string]string{"error": "could not resume attempt"})
			return
		}
		write(w, 200, resp)
		return
	}
	var used int
	if err = tx.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM exam_attempts a JOIN exam_sessions es ON es.id=a.session_id WHERE es.exam_id=$1 AND es.student_id=$2`, examID, student.UserID).Scan(&used); err != nil {
		write(w, 500, map[string]string{"error": "could not verify attempts"})
		return
	}
	if used >= limit {
		write(w, 409, map[string]string{"error": "attempt limit reached"})
		return
	}
	sessionID := uuid.New()
	attemptID := uuid.New()
	deadline := now.Add(time.Duration(duration) * time.Minute)
	if _, err = tx.ExecContext(r.Context(), `INSERT INTO exam_sessions(id,exam_id,student_id,started_at,server_deadline,status) VALUES($1,$2,$3,$4,$5,'active')`, sessionID, examID, student.UserID, now, deadline); err != nil {
		s.Logger.Error("create exam session failed", "error", err, "examId", examID, "studentId", student.UserID)
		write(w, 500, map[string]string{"error": "could not create exam session"})
		return
	}
	if _, err = tx.ExecContext(r.Context(), `INSERT INTO exam_attempts(id,session_id,attempt_number,status,started_at,expires_at) VALUES($1,$2,$3,'in_progress',$4,$5)`, attemptID, sessionID, used+1, now, deadline); err != nil {
		s.Logger.Error("create attempt failed", "error", err, "examId", examID, "studentId", student.UserID)
		write(w, 500, map[string]string{"error": "could not create attempt"})
		return
	}
	if _, err = tx.ExecContext(r.Context(), `INSERT INTO proctoring_sessions(id,attempt_id) VALUES($1,$2)`, uuid.New(), attemptID); err != nil {
		s.Logger.Error("create proctoring session failed", "error", err, "attemptId", attemptID)
		write(w, 500, map[string]string{"error": "could not create proctoring session"})
		return
	}
	if err = s.persistQuestionSet(r.Context(), tx, attemptID, examID, rq, ro); err != nil {
		s.Logger.Error("generate question set failed", "error", err, "attemptId", attemptID)
		write(w, 500, map[string]string{"error": "could not generate question set"})
		return
	}
	resp, err := s.attemptPayloadTx(r.Context(), tx, attemptID, now)
	if err != nil {
		s.Logger.Error("load attempt failed", "error", err, "attemptId", attemptID)
		write(w, 500, map[string]string{"error": "could not load attempt"})
		return
	}
	resp.ExamTitle = title
	resp.AllowReview = allowReview
	if err = tx.Commit(); err != nil {
		write(w, 500, map[string]string{"error": "could not commit attempt"})
		return
	}
	s.audit(r.Context(), student.UserID, "ATTEMPT_CREATED", "exam_attempts", attemptID, "SUCCESS", map[string]interface{}{"examId": examID.String()})
	write(w, 201, resp)
}

func (s *Server) getAttempt(w http.ResponseWriter, r *http.Request) {
	attemptID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid attempt id"})
		return
	}
	c := claimsOf(r)
	if !s.canAccessAttempt(r.Context(), attemptID, c) {
		write(w, 403, map[string]string{"error": "attempt access denied"})
		return
	}
	resp, err := s.attemptPayload(r.Context(), attemptID, time.Now())
	if err != nil {
		write(w, 404, map[string]string{"error": "attempt not found"})
		return
	}
	if resp.Status == "in_progress" && time.Now().After(resp.ExpiresAt) {
		_, _ = s.autoSubmitAttempt(r.Context(), attemptID)
		resp, err = s.attemptPayload(r.Context(), attemptID, time.Now())
	}
	if err != nil {
		write(w, 500, map[string]string{"error": "could not load attempt"})
		return
	}
	write(w, 200, resp)
}
func (s *Server) saveAnswer(w http.ResponseWriter, r *http.Request) {
	attemptID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid attempt id"})
		return
	}
	in := answerInput{}
	if !decode(w, r, &in) {
		return
	}
	if in.QuestionID == "" {
		in.QuestionID = r.PathValue("questionId")
	}
	qid, err := parseUUID(in.QuestionID)
	if err != nil {
		write(w, 400, map[string]string{"error": "valid questionId is required"})
		return
	}
	c := claimsOf(r)
	tx, err := s.DB.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		write(w, 500, map[string]string{"error": "could not save answer"})
		return
	}
	defer tx.Rollback()
	var studentID, status string
	var expires time.Time
	err = tx.QueryRowContext(r.Context(), `SELECT es.student_id,a.status,a.expires_at FROM exam_attempts a JOIN exam_sessions es ON es.id=a.session_id WHERE a.id=$1 FOR UPDATE`, attemptID).Scan(&studentID, &status, &expires)
	if err != nil {
		write(w, 404, map[string]string{"error": "attempt not found"})
		return
	}
	if studentID != c.UserID {
		write(w, 403, map[string]string{"error": "attempt access denied"})
		return
	}
	if status != "in_progress" {
		write(w, 409, map[string]string{"error": "attempt is no longer editable"})
		return
	}
	if !time.Now().Before(expires) {
		_, _ = s.finalizeAttemptTx(r.Context(), tx, attemptID, "auto_submitted")
		_ = tx.Commit()
		write(w, 409, map[string]string{"error": "attempt expired and was auto-submitted"})
		return
	}
	var belongs bool
	_ = tx.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM attempt_questions WHERE attempt_id=$1 AND question_id=$2)`, attemptID, qid).Scan(&belongs)
	if !belongs {
		write(w, 400, map[string]string{"error": "question does not belong to attempt"})
		return
	}
	values, _ := json.Marshal(in.Values)
	_, err = tx.ExecContext(r.Context(), `INSERT INTO student_answers(id,attempt_id,question_id,values,text_answer,marked_for_review,saved_at) VALUES($1,$2,$3,$4,$5,$6,now()) ON CONFLICT(attempt_id,question_id) DO UPDATE SET values=EXCLUDED.values,text_answer=EXCLUDED.text_answer,marked_for_review=EXCLUDED.marked_for_review,saved_at=now()`, uuid.New(), attemptID, qid, values, in.TextAnswer, in.MarkedForReview)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not persist answer"})
		return
	}
	if err = s.auditTx(r.Context(), tx, studentID, "ANSWER_SAVED", "student_answers", attemptID, "SUCCESS", map[string]interface{}{"questionId": qid.String()}); err != nil {
		s.Logger.Warn("audit answer save failed", "error", err)
	}
	if err = tx.Commit(); err != nil {
		write(w, 500, map[string]string{"error": "could not commit answer"})
		return
	}
	write(w, 200, map[string]interface{}{"status": "saved", "savedAt": time.Now()})
}

func (s *Server) submitAttempt(w http.ResponseWriter, r *http.Request) {
	attemptID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid attempt id"})
		return
	}
	c := claimsOf(r)
	tx, err := s.DB.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		write(w, 500, map[string]string{"error": "could not submit attempt"})
		return
	}
	defer tx.Rollback()
	var studentID, status string
	var expires time.Time
	err = tx.QueryRowContext(r.Context(), `SELECT es.student_id,a.status,a.expires_at FROM exam_attempts a JOIN exam_sessions es ON es.id=a.session_id WHERE a.id=$1 FOR UPDATE`, attemptID).Scan(&studentID, &status, &expires)
	if err != nil {
		write(w, 404, map[string]string{"error": "attempt not found"})
		return
	}
	if studentID != c.UserID {
		examID, examErr := s.attemptExam(r.Context(), attemptID)
		if examErr != nil || !s.canManageExam(r.Context(), c, examID) {
			write(w, 403, map[string]string{"error": "attempt access denied"})
			return
		}
	}
	if status == "submitted" || status == "auto_submitted" || status == "expired" {
		res, err := s.resultForAttemptTx(r.Context(), tx, attemptID)
		if err != nil {
			write(w, 500, map[string]string{"error": "could not load result"})
			return
		}
		_ = tx.Commit()
		write(w, 200, res)
		return
	}
	finalStatus := "submitted"
	if !time.Now().Before(expires) {
		finalStatus = "auto_submitted"
	}
	res, err := s.finalizeAttemptTx(r.Context(), tx, attemptID, finalStatus)
	if err != nil {
		s.Logger.Error("grade attempt failed", "error", err, "attemptId", attemptID)
		write(w, 500, map[string]string{"error": "could not grade attempt"})
		return
	}
	if err = tx.Commit(); err != nil {
		write(w, 500, map[string]string{"error": "could not commit submission"})
		return
	}
	s.audit(r.Context(), studentID, map[string]string{"submitted": "EXAM_SUBMITTED", "auto_submitted": "EXAM_AUTO_SUBMITTED"}[finalStatus], "exam_attempts", attemptID, "SUCCESS", map[string]interface{}{})
	write(w, 200, res)
}

func (s *Server) attemptResult(w http.ResponseWriter, r *http.Request) {
	attemptID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid attempt id"})
		return
	}
	c := claimsOf(r)
	if !s.canAccessAttempt(r.Context(), attemptID, c) {
		write(w, 403, map[string]string{"error": "result access denied"})
		return
	}
	res, err := s.resultForAttempt(r.Context(), attemptID)
	if err != nil {
		write(w, 404, map[string]string{"error": "result not found"})
		return
	}
	if c.UserID != s.attemptStudent(r.Context(), attemptID) && !s.canAccessAttempt(r.Context(), attemptID, c) {
		write(w, 403, map[string]string{"error": "result access denied"})
		return
	}
	if c.UserID == s.attemptStudent(r.Context(), attemptID) && !res.Published {
		write(w, 403, map[string]string{"error": "result is not published"})
		return
	}
	write(w, 200, res)
}
func (s *Server) instructorResults(w http.ResponseWriter, r *http.Request) {
	examID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid exam id"})
		return
	}
	c := claimsOf(r)
	if !s.canAccessExam(r.Context(), c, examID) {
		write(w, 403, map[string]string{"error": "exam scope denied"})
		return
	}
	rows, err := s.DB.QueryContext(r.Context(), `SELECT re.id,re.attempt_id,e.title,re.score,re.maximum,re.percentage,re.grade,re.correct_count,re.incorrect_count,re.unanswered_count,re.duration_seconds,re.published_at,a.submitted_at FROM results re JOIN exam_attempts a ON a.id=re.attempt_id JOIN exam_sessions es ON es.id=a.session_id JOIN exams e ON e.id=es.exam_id WHERE e.id=$1 ORDER BY a.submitted_at DESC`, examID)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not load results"})
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, aid, title, grade string
		var score, maxi, pct float64
		var correct, incorrect, unanswered, duration int
		var published, submitted sql.NullTime
		if rows.Scan(&id, &aid, &title, &score, &maxi, &pct, &grade, &correct, &incorrect, &unanswered, &duration, &published, &submitted) == nil {
			out = append(out, map[string]interface{}{"id": id, "attemptId": aid, "examTitle": title, "score": score, "maximum": maxi, "percentage": pct, "grade": grade, "correct": correct, "incorrect": incorrect, "unanswered": unanswered, "durationSeconds": duration, "published": published.Valid, "submittedAt": timeValue(submitted)})
		}
	}
	write(w, 200, out)
}
func (s *Server) publishResult(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid result id"})
		return
	}
	c := claimsOf(r)
	var examID uuid.UUID
	if err := s.DB.QueryRowContext(r.Context(), `SELECT es.exam_id FROM results re JOIN exam_attempts a ON a.id=re.attempt_id JOIN exam_sessions es ON es.id=a.session_id WHERE re.id=$1`, id).Scan(&examID); err != nil || !s.canManageExam(r.Context(), c, examID) {
		write(w, 403, map[string]string{"error": "result scope denied"})
		return
	}
	res, err := s.DB.ExecContext(r.Context(), `UPDATE results SET published_at=now() WHERE id=$1`, id)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not publish result"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		write(w, 404, map[string]string{"error": "result not found or not owned"})
		return
	}
	s.audit(r.Context(), c.UserID, "RESULT_PUBLISHED", "results", id, "SUCCESS", map[string]interface{}{})
	write(w, 200, map[string]interface{}{"id": id.String(), "published": true})
}

func (s *Server) persistQuestionSet(ctx context.Context, tx *sql.Tx, attemptID, examID uuid.UUID, randomQuestions, randomOptions bool) error {
	seed := attemptID.String()
	type bundle struct {
		id      string
		points  float64
		typ     string
		options []option
	}
	rows, err := tx.QueryContext(ctx, `SELECT eq.question_id,eq.points,q.type,qo.option_key,qo.option_text FROM exam_questions eq JOIN questions q ON q.id=eq.question_id LEFT JOIN question_options qo ON qo.question_id=q.id WHERE eq.exam_id=$1 ORDER BY eq.position,qo.option_key`, examID)
	if err != nil {
		return err
	}
	defer rows.Close()
	byID := map[string]*bundle{}
	order := []string{}
	for rows.Next() {
		var id, typ string
		var points float64
		var key, text sql.NullString
		if err := rows.Scan(&id, &points, &typ, &key, &text); err != nil {
			return err
		}
		b, ok := byID[id]
		if !ok {
			b = &bundle{id: id, points: points, typ: typ}
			byID[id] = b
			order = append(order, id)
		}
		if key.Valid {
			b.options = append(b.options, option{Key: key.String, Text: text.String})
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if randomQuestions {
		sort.Slice(order, func(i, j int) bool { return hashOrder(seed+order[i]) < hashOrder(seed+order[j]) })
	}
	for position, id := range order {
		b := byID[id]
		if randomOptions {
			sort.Slice(b.options, func(i, j int) bool { return hashOrder(seed+b.options[i].Key) < hashOrder(seed+b.options[j].Key) })
		}
		encoded, _ := json.Marshal(b.options)
		if _, err := tx.ExecContext(ctx, `INSERT INTO attempt_questions(attempt_id,question_id,position,points,option_order) VALUES($1,$2,$3,$4,$5)`, attemptID, id, position+1, b.points, encoded); err != nil {
			return err
		}
	}
	return nil
}
func hashOrder(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func (s *Server) attemptPayload(ctx context.Context, attemptID uuid.UUID, now time.Time) (attemptResponse, error) {
	return s.attemptPayloadTx(ctx, nil, attemptID, now)
}
func (s *Server) attemptPayloadTx(ctx context.Context, tx *sql.Tx, attemptID uuid.UUID, now time.Time) (attemptResponse, error) {
	var out attemptResponse
	out.ID = attemptID.String()
	out.ServerTime = now
	var q interface {
		QueryRowContext(context.Context, string, ...interface{}) *sql.Row
		QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	} = s.DB
	if tx != nil {
		q = tx
	}
	var examID, title, status string
	var started, expires time.Time
	var allow bool
	row := q.QueryRowContext(ctx, `SELECT es.exam_id,e.title,a.status,a.started_at,a.expires_at,e.allow_review FROM exam_attempts a JOIN exam_sessions es ON es.id=a.session_id JOIN exams e ON e.id=es.exam_id WHERE a.id=$1`, attemptID)
	if err := row.Scan(&examID, &title, &status, &started, &expires, &allow); err != nil {
		return out, err
	}
	out.ExamID = examID
	out.ExamTitle = title
	out.Status = status
	out.StartedAt = started
	out.ExpiresAt = expires
	out.AllowReview = allow
	rows, err := q.QueryContext(ctx, `SELECT aq.question_id,aq.position,q.type,q.prompt,aq.points,aq.option_order FROM attempt_questions aq JOIN questions q ON q.id=aq.question_id WHERE aq.attempt_id=$1 ORDER BY aq.position`, attemptID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, typ, prompt string
		var pos int
		var points float64
		var optsJSON []byte
		if err = rows.Scan(&id, &pos, &typ, &prompt, &points, &optsJSON); err != nil {
			return out, err
		}
		var opts []option
		_ = json.Unmarshal(optsJSON, &opts)
		out.Questions = append(out.Questions, examQuestion{ID: id, Position: pos, Type: typ, Prompt: prompt, Points: points, Options: opts})
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	out.Answers = map[string]answerInput{}
	arows, err := q.QueryContext(ctx, `SELECT question_id,values,text_answer,marked_for_review FROM student_answers WHERE attempt_id=$1`, attemptID)
	if err != nil {
		return out, err
	}
	defer arows.Close()
	for arows.Next() {
		var qid string
		var valuesJSON []byte
		var text string
		var review bool
		if arows.Scan(&qid, &valuesJSON, &text, &review) == nil {
			var vals []string
			_ = json.Unmarshal(valuesJSON, &vals)
			out.Answers[qid] = answerInput{QuestionID: qid, Values: vals, TextAnswer: text, MarkedForReview: review}
		}
	}
	return out, arows.Err()
}

func (s *Server) finalizeAttemptTx(ctx context.Context, tx *sql.Tx, attemptID uuid.UUID, status string) (resultResponse, error) {
	if status != "submitted" && status != "auto_submitted" && status != "expired" {
		return resultResponse{}, errors.New("invalid final status")
	}
	var studentID, examTitle, examID string
	var started, submitted time.Time
	var negative float64
	err := tx.QueryRowContext(ctx, `SELECT es.student_id,e.title,e.id,e.negative_marking,a.started_at,COALESCE(a.submitted_at,now()) FROM exam_attempts a JOIN exam_sessions es ON es.id=a.session_id JOIN exams e ON e.id=es.exam_id WHERE a.id=$1`, attemptID).Scan(&studentID, &examTitle, &examID, &negative, &started, &submitted)
	if err != nil {
		return resultResponse{}, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT aq.question_id,q.type,aq.points,COALESCE(array_agg(qo.option_key) FILTER(WHERE qo.is_correct),'{}') FROM attempt_questions aq JOIN questions q ON q.id=aq.question_id LEFT JOIN question_options qo ON qo.question_id=q.id WHERE aq.attempt_id=$1 GROUP BY aq.question_id,q.type,aq.points,aq.position ORDER BY aq.position`, attemptID)
	if err != nil {
		return resultResponse{}, err
	}
	defer rows.Close()
	qs := []grading.Question{}
	for rows.Next() {
		var id, typ string
		var points float64
		var correct pq.StringArray
		if rows.Scan(&id, &typ, &points, &correct) != nil {
			continue
		}
		qs = append(qs, grading.Question{ID: id, Type: typ, Points: points, Correct: correct})
	}
	answersRows, err := tx.QueryContext(ctx, `SELECT question_id,values FROM student_answers WHERE attempt_id=$1`, attemptID)
	if err != nil {
		return resultResponse{}, err
	}
	defer answersRows.Close()
	answers := []grading.Answer{}
	for answersRows.Next() {
		var qid string
		var valsJSON []byte
		var vals []string
		if answersRows.Scan(&qid, &valsJSON) == nil {
			_ = json.Unmarshal(valsJSON, &vals)
			answers = append(answers, grading.Answer{QuestionID: qid, Values: vals})
		}
	}
	outcome := grading.Grade(qs, answers, negative)
	pct := 0.0
	if outcome.Maximum > 0 {
		pct = math.Round(outcome.Earned/outcome.Maximum*10000) / 100
	}
	grade := gradeFor(pct)
	now := time.Now()
	_, err = tx.ExecContext(ctx, `UPDATE exam_attempts SET status=$1,submitted_at=$2,score=$3,percentage=$4,grade=$5 WHERE id=$6 AND status='in_progress'`, status, now, outcome.Earned, pct, grade, attemptID)
	if err != nil {
		return resultResponse{}, err
	}
	var resultID string
	err = tx.QueryRowContext(ctx, `INSERT INTO results(id,attempt_id,score,maximum,percentage,grade,correct_count,incorrect_count,unanswered_count,duration_seconds) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(attempt_id) DO UPDATE SET score=EXCLUDED.score,maximum=EXCLUDED.maximum,percentage=EXCLUDED.percentage,grade=EXCLUDED.grade,correct_count=EXCLUDED.correct_count,incorrect_count=EXCLUDED.incorrect_count,unanswered_count=EXCLUDED.unanswered_count,duration_seconds=EXCLUDED.duration_seconds RETURNING id`, uuid.New(), attemptID, outcome.Earned, outcome.Maximum, pct, grade, outcome.Correct, outcome.Incorrect, outcome.Unanswered, int(time.Since(started).Seconds())).Scan(&resultID)
	if err != nil {
		return resultResponse{}, err
	}
	_ = s.auditTx(ctx, tx, studentID, map[string]string{"submitted": "RESULT_GENERATED", "auto_submitted": "EXAM_AUTO_SUBMITTED", "expired": "EXAM_AUTO_SUBMITTED"}[status], "exam_attempts", attemptID, "SUCCESS", map[string]interface{}{"resultId": resultID})
	return resultResponse{ID: resultID, AttemptID: attemptID.String(), ExamTitle: examTitle, Score: outcome.Earned, Maximum: outcome.Maximum, Percentage: pct, Grade: grade, Correct: outcome.Correct, Incorrect: outcome.Incorrect, Unanswered: outcome.Unanswered, SubmittedAt: now, DurationSeconds: int(time.Since(started).Seconds()), Published: false}, nil
}
func (s *Server) resultForAttempt(ctx context.Context, attemptID uuid.UUID) (resultResponse, error) {
	return s.resultForAttemptTx(ctx, nil, attemptID)
}
func (s *Server) resultForAttemptTx(ctx context.Context, tx *sql.Tx, attemptID uuid.UUID) (resultResponse, error) {
	var out resultResponse
	var q interface {
		QueryRowContext(context.Context, string, ...interface{}) *sql.Row
	} = s.DB
	if tx != nil {
		q = tx
	}
	var id, examTitle, grade string
	var score, maxi, pct float64
	var correct, incorrect, unanswered, duration int
	var published, submitted sql.NullTime
	err := q.QueryRowContext(ctx, `SELECT r.id,r.score,r.maximum,r.percentage,r.grade,r.correct_count,r.incorrect_count,r.unanswered_count,r.duration_seconds,r.published_at,a.submitted_at,e.title FROM results r JOIN exam_attempts a ON a.id=r.attempt_id JOIN exam_sessions es ON es.id=a.session_id JOIN exams e ON e.id=es.exam_id WHERE r.attempt_id=$1`, attemptID).Scan(&id, &score, &maxi, &pct, &grade, &correct, &incorrect, &unanswered, &duration, &published, &submitted, &examTitle)
	if err != nil {
		return out, err
	}
	out = resultResponse{ID: id, AttemptID: attemptID.String(), ExamTitle: examTitle, Score: score, Maximum: maxi, Percentage: pct, Grade: grade, Correct: correct, Incorrect: incorrect, Unanswered: unanswered, Published: published.Valid, SubmittedAt: submitted.Time, DurationSeconds: duration}
	return out, nil
}
func (s *Server) autoSubmitAttempt(ctx context.Context, id uuid.UUID) (resultResponse, error) {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return resultResponse{}, err
	}
	defer tx.Rollback()
	var status string
	var expires time.Time
	if err = tx.QueryRowContext(ctx, `SELECT status,expires_at FROM exam_attempts WHERE id=$1 FOR UPDATE`, id).Scan(&status, &expires); err != nil {
		return resultResponse{}, err
	}
	if status != "in_progress" || time.Now().Before(expires) {
		return s.resultForAttemptTx(ctx, tx, id)
	}
	res, err := s.finalizeAttemptTx(ctx, tx, id, "auto_submitted")
	if err != nil {
		return resultResponse{}, err
	}
	if err = tx.Commit(); err != nil {
		return resultResponse{}, err
	}
	return res, nil
}
func (s *Server) AutoSubmitExpired(ctx context.Context) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id FROM exam_attempts WHERE status='in_progress' AND expires_at<=now() LIMIT 100`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			if _, err := s.autoSubmitAttempt(ctx, uuid.MustParse(id)); err != nil && s.Logger != nil {
				s.Logger.Warn("auto-submit failed", "attemptId", id, "error", err)
			}
		}
	}
}

func (s *Server) attemptStudent(ctx context.Context, id uuid.UUID) string {
	var student string
	_ = s.DB.QueryRowContext(ctx, `SELECT es.student_id FROM exam_attempts a JOIN exam_sessions es ON es.id=a.session_id WHERE a.id=$1`, id).Scan(&student)
	return student
}
func (s *Server) audit(ctx context.Context, userID, action, resource string, id uuid.UUID, result string, metadata map[string]interface{}) {
	_ = s.auditExec(ctx, s.DB, userID, action, resource, id, result, metadata)
}
func (s *Server) auditTx(ctx context.Context, tx *sql.Tx, userID, action, resource string, id uuid.UUID, result string, metadata map[string]interface{}) error {
	return s.auditExec(ctx, tx, userID, action, resource, id, result, metadata)
}
func (s *Server) auditExec(ctx context.Context, exec sqlExecer, userID, action, resource string, id uuid.UUID, result string, metadata map[string]interface{}) error {
	data, _ := json.Marshal(metadata)
	_, err := exec.ExecContext(ctx, `INSERT INTO audit_logs(id,user_id,action,resource_type,resource_id,result,metadata) VALUES($1,$2,$3,$4,$5,$6,$7)`, uuid.New(), userID, action, resource, id, result, data)
	return err
}

func validateExam(in createExamRequest) error {
	if strings.TrimSpace(in.Title) == "" {
		return errors.New("title is required")
	}
	if in.DurationMinutes < 1 || in.DurationMinutes > 1440 {
		return errors.New("duration must be between 1 and 1440 minutes")
	}
	if in.AttemptLimit < 0 || in.AttemptLimit > 20 {
		return errors.New("attemptLimit must be between 1 and 20")
	}
	if in.PassingScore < 0 || in.PassingScore > 100 {
		return errors.New("passingScore must be between 0 and 100")
	}
	if in.NegativeMarking < 0 {
		return errors.New("negativeMarking cannot be negative")
	}
	if in.StartAt != nil && in.EndAt != nil && in.EndAt.Before(*in.StartAt) {
		return errors.New("endAt must be after startAt")
	}
	return nil
}
func validQuestionType(t string) bool {
	switch t {
	case "multiple_choice", "multiple_select", "true_false", "short_answer", "essay":
		return true
	}
	return false
}
func defaultVisibility(v string) string {
	if v == "published" {
		return v
	}
	return "not_published"
}
func parseUUID(v string) (uuid.UUID, error) { return uuid.Parse(v) }
func decode(w http.ResponseWriter, r *http.Request, d interface{}) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(d); err != nil {
		write(w, 400, map[string]string{"error": "invalid request body"})
		return false
	}
	return true
}
func write(w http.ResponseWriter, status int, v interface{}) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func timeValue(v sql.NullTime) interface{} {
	if v.Valid {
		return v.Time
	}
	return nil
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func gradeFor(p float64) string {
	switch {
	case p >= 90:
		return "A"
	case p >= 80:
		return "B"
	case p >= 70:
		return "C"
	case p >= 60:
		return "D"
	default:
		return "F"
	}
}
