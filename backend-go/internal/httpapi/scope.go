package httpapi

import (
	"context"
	"database/sql"

	"github.com/A181-CODER/HNU-TEST-/backend-go/internal/auth"
	"github.com/google/uuid"
)

func hasRole(c *auth.Claims, role string) bool { return c != nil && auth.HasRole(c.Roles, role) }

func (s *Server) courseForExam(ctx context.Context, examID uuid.UUID) (uuid.UUID, error) {
	var courseID uuid.UUID
	err := s.DB.QueryRowContext(ctx, `SELECT course_id FROM exams WHERE id=$1 AND deleted_at IS NULL`, examID).Scan(&courseID)
	return courseID, err
}

func (s *Server) userHasRole(ctx context.Context, userID, role string) bool {
	var exists bool
	_ = s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id WHERE ur.user_id=$1 AND r.slug=$2)`, userID, role).Scan(&exists)
	return exists
}

func (s *Server) canAccessCourse(ctx context.Context, c *auth.Claims, courseID uuid.UUID) bool {
	if c == nil || s.DB == nil {
		return false
	}
	if hasRole(c, "super_admin") {
		return true
	}
	if hasRole(c, "student") {
		var enrolled bool
		_ = s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM course_students WHERE course_id=$1 AND student_id=$2)`, courseID, c.UserID).Scan(&enrolled)
		return enrolled
	}
	if hasRole(c, "proctor") {
		return false
	}
	var scoped bool
	_ = s.DB.QueryRowContext(ctx, `SELECT EXISTS(
		SELECT 1 FROM courses co
		LEFT JOIN departments d ON d.id=co.department_id
		LEFT JOIN faculties f ON f.id=d.faculty_id
		JOIN resource_memberships rm ON rm.user_id=$1
		WHERE co.id=$2 AND rm.role_slug IN ('instructor','university_admin') AND
			(rm.course_id=co.id OR rm.department_id=co.department_id OR rm.faculty_id=f.id OR rm.university_id=f.university_id)
	)`, c.UserID, courseID).Scan(&scoped)
	return scoped && (hasRole(c, "instructor") || hasRole(c, "university_admin"))
}

func (s *Server) canManageCourse(ctx context.Context, c *auth.Claims, courseID uuid.UUID) bool {
	if hasRole(c, "super_admin") {
		return true
	}
	return s.canAccessCourse(ctx, c, courseID) && (hasRole(c, "instructor") || hasRole(c, "university_admin"))
}

func (s *Server) canAccessExam(ctx context.Context, c *auth.Claims, examID uuid.UUID) bool {
	if c == nil || s.DB == nil {
		return false
	}
	if hasRole(c, "super_admin") {
		return true
	}
	if hasRole(c, "student") {
		courseID, err := s.courseForExam(ctx, examID)
		return err == nil && s.canAccessCourse(ctx, c, courseID)
	}
	if hasRole(c, "proctor") {
		var assigned bool
		_ = s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM exam_proctors WHERE exam_id=$1 AND proctor_id=$2)`, examID, c.UserID).Scan(&assigned)
		return assigned
	}
	courseID, err := s.courseForExam(ctx, examID)
	return err == nil && s.canAccessCourse(ctx, c, courseID) && (hasRole(c, "instructor") || hasRole(c, "university_admin"))
}

func (s *Server) canManageExam(ctx context.Context, c *auth.Claims, examID uuid.UUID) bool {
	return s.canAccessExam(ctx, c, examID) && (hasRole(c, "super_admin") || hasRole(c, "instructor") || hasRole(c, "university_admin"))
}

func (s *Server) canAccessAttempt(ctx context.Context, id uuid.UUID, c *auth.Claims) bool {
	if c == nil || s.DB == nil {
		return false
	}
	var student string
	var examID uuid.UUID
	if s.DB.QueryRowContext(ctx, `SELECT es.student_id,es.exam_id FROM exam_attempts a JOIN exam_sessions es ON es.id=a.session_id WHERE a.id=$1`, id).Scan(&student, &examID) != nil {
		return false
	}
	if student == c.UserID && hasRole(c, "student") {
		return true
	}
	return s.canAccessExam(ctx, c, examID)
}

func (s *Server) attemptExam(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	var examID uuid.UUID
	err := s.DB.QueryRowContext(ctx, `SELECT es.exam_id FROM exam_attempts a JOIN exam_sessions es ON es.id=a.session_id WHERE a.id=$1`, id).Scan(&examID)
	return examID, err
}

func (s *Server) resourceSummaryScope(ctx context.Context, c *auth.Claims) (string, []interface{}) {
	if c == nil {
		return "FALSE", nil
	}
	if hasRole(c, "super_admin") {
		return "TRUE", nil
	}
	return `EXISTS (SELECT 1 FROM resource_memberships rm WHERE rm.user_id=$1 AND (rm.university_id=u.id OR rm.faculty_id=f.id OR rm.department_id=d.id))`, []interface{}{c.UserID}
}

func nullUUID(v uuid.UUID) interface{} {
	if v == uuid.Nil {
		return nil
	}
	return v
}
func scanOptionalUUID(row *sql.Row) (uuid.UUID, error) {
	var id uuid.UUID
	err := row.Scan(&id)
	return id, err
}
