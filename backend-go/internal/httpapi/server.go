package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/A181-CODER/HNU-TEST-/backend-go/internal/auth"
	"github.com/google/uuid"
)

type Server struct {
	DB     *sql.DB
	Auth   auth.Service
	Logger *slog.Logger
	CORS   string
}
type contextKey string

const claimsKey contextKey = "claims"

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

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /ready", s.ready)
	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.Handle("GET /api/v1/me", s.requireRole(http.HandlerFunc(s.me)))
	mux.Handle("GET /api/v1/exams", s.requireRole(http.HandlerFunc(s.exams)))
	mux.Handle("POST /api/v1/exams", s.requireAnyRole([]string{"super_admin", "university_admin", "instructor"})(http.HandlerFunc(s.createExam)))
	mux.Handle("POST /api/v1/exams/{id}/start", s.requireRole(http.HandlerFunc(s.startExam)))
	mux.Handle("POST /api/v1/attempts/{id}/answers", s.requireRole(http.HandlerFunc(s.saveAnswer)))
	mux.Handle("POST /api/v1/attempts/{id}/submit", s.requireRole(http.HandlerFunc(s.submitAttempt)))
	return s.middleware(mux)
}
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self' http://localhost:8080 http://localhost:8000; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'")
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
	write(w, 200, map[string]string{"status": "ok", "service": "hnu-api"})
}
func (s *Server) ready(w http.ResponseWriter, _ *http.Request) {
	if s.DB != nil {
		if err := s.DB.Ping(); err != nil {
			write(w, 503, map[string]string{"status": "not_ready"})
			return
		}
	}
	write(w, 200, map[string]string{"status": "ready"})
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in loginRequest
	if json.NewDecoder(r.Body).Decode(&in) != nil || !strings.Contains(in.Email, "@") {
		write(w, 400, map[string]string{"error": "invalid credentials format"})
		return
	}
	if s.DB == nil {
		write(w, 503, map[string]string{"error": "database is not configured"})
		return
	}
	var id, name, email, hash string
	var rolesJSON []byte
	err := s.DB.QueryRowContext(r.Context(), `SELECT u.id,u.display_name,u.email,u.password_hash,COALESCE(json_agg(r.slug) FILTER (WHERE r.slug IS NOT NULL),'[]') FROM users u LEFT JOIN user_roles ur ON ur.user_id=u.id LEFT JOIN roles r ON r.id=ur.role_id WHERE lower(u.email)=lower($1) AND u.deleted_at IS NULL GROUP BY u.id`, in.Email).Scan(&id, &name, &email, &hash, &rolesJSON)
	if err != nil || !auth.CheckPassword(hash, in.Password) {
		write(w, 401, map[string]string{"error": "invalid email or password"})
		return
	}
	var roles []string
	_ = json.Unmarshal(rolesJSON, &roles)
	token, err := s.Auth.Issue(id, roles)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not issue session"})
		return
	}
	_, _ = s.DB.ExecContext(r.Context(), `INSERT INTO audit_logs(id,user_id,action,result,metadata) VALUES($1,$2,'LOGIN','SUCCESS','{}')`, uuid.New(), id)
	write(w, 200, loginResponse{AccessToken: token, User: userResponse{ID: id, Name: name, Email: email, Roles: roles}})
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
	return s.Auth.Parse(r.Header.Get("Authorization"))
}
func contextWithClaims(r *http.Request, c *auth.Claims) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), claimsKey, c))
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Context().Value(claimsKey).(*auth.Claims)
	write(w, 200, map[string]interface{}{"id": c.UserID, "roles": c.Roles})
}
func (s *Server) exams(w http.ResponseWriter, _ *http.Request) {
	if s.DB == nil {
		write(w, 200, []interface{}{})
		return
	}
	rows, err := s.DB.Query(`SELECT id,title,course_code,status,start_at,end_at,duration_minutes FROM exams WHERE deleted_at IS NULL ORDER BY start_at NULLS LAST`)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not load exams"})
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, title, course, status string
		var start, end sql.NullTime
		var duration int
		if rows.Scan(&id, &title, &course, &status, &start, &end, &duration) == nil {
			out = append(out, map[string]interface{}{"id": id, "title": title, "courseCode": course, "status": status, "startAt": start.Time, "endAt": end.Time, "durationMinutes": duration})
		}
	}
	write(w, 200, out)
}
func (s *Server) createExam(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Title           string `json:"title"`
		CourseCode      string `json:"courseCode"`
		DurationMinutes int    `json:"durationMinutes"`
		Instructions    string `json:"instructions"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Title) == "" || in.DurationMinutes < 1 {
		write(w, 400, map[string]string{"error": "title and positive duration are required"})
		return
	}
	c, _ := r.Context().Value(claimsKey).(*auth.Claims)
	id := uuid.New()
	if s.DB == nil {
		write(w, 503, map[string]string{"error": "database is not configured"})
		return
	}
	_, err := s.DB.ExecContext(r.Context(), `INSERT INTO exams(id,created_by,title,course_code,duration_minutes,instructions,status) VALUES($1,$2,$3,$4,$5,$6,'draft')`, id, c.UserID, in.Title, in.CourseCode, in.DurationMinutes, in.Instructions)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not create exam"})
		return
	}
	write(w, 201, map[string]interface{}{"id": id, "status": "draft"})
}
func (s *Server) startExam(w http.ResponseWriter, r *http.Request) {
	write(w, 501, map[string]string{"error": "attempt lifecycle is defined; scheduling and preflight integration is next"})
}
func (s *Server) saveAnswer(w http.ResponseWriter, r *http.Request) {
	write(w, 501, map[string]string{"error": "answer persistence endpoint is reserved for the next exam-engine slice"})
}
func (s *Server) submitAttempt(w http.ResponseWriter, r *http.Request) {
	write(w, 501, map[string]string{"error": "server-side submission endpoint is reserved for the next exam-engine slice"})
}
func write(w http.ResponseWriter, status int, v interface{}) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
