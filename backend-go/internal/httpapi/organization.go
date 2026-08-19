package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/A181-CODER/HNU-TEST-/backend-go/internal/auth"
	"github.com/google/uuid"
)

type assignmentRequest struct {
	UserID       string `json:"userId"`
	Role         string `json:"role"`
	UniversityID string `json:"universityId"`
	FacultyID    string `json:"facultyId"`
	DepartmentID string `json:"departmentId"`
	CourseID     string `json:"courseId"`
}
type courseEnrollmentRequest struct {
	UserID string `json:"userId"`
}
type examProctorRequest struct {
	UserID string `json:"userId"`
}

func parseOptionalUUID(value string) (interface{}, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	return uuid.Parse(value)
}

func (s *Server) organizationTree(w http.ResponseWriter, r *http.Request) {
	c := claimsOf(r)
	if c == nil {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	universities, err := s.DB.QueryContext(r.Context(), `SELECT u.id,u.name,u.code,COUNT(DISTINCT f.id) FROM universities u LEFT JOIN faculties f ON f.university_id=u.id GROUP BY u.id,u.name,u.code ORDER BY u.name`)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not load organization"})
		return
	}
	defer universities.Close()
	out := []map[string]interface{}{}
	for universities.Next() {
		var id, name, code string
		var facultyCount int
		if universities.Scan(&id, &name, &code, &facultyCount) != nil {
			continue
		}
		if !s.canAccessUniversity(r.Context(), c, uuid.MustParse(id)) {
			continue
		}
		faculties := s.facultiesForUniversity(r.Context(), c, uuid.MustParse(id))
		out = append(out, map[string]interface{}{"id": id, "name": name, "code": code, "facultyCount": facultyCount, "faculties": faculties})
	}
	write(w, 200, map[string]interface{}{"universities": out, "viewer": map[string]interface{}{"userId": c.UserID, "roles": c.Roles}})
}

func (s *Server) canAccessUniversity(ctx context.Context, c *auth.Claims, id uuid.UUID) bool {
	if hasRole(c, "super_admin") {
		return true
	}
	var ok bool
	if err := s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM resource_memberships rm WHERE rm.user_id=$1 AND (rm.university_id=$2 OR rm.faculty_id IN (SELECT id FROM faculties WHERE university_id=$2) OR rm.department_id IN (SELECT d.id FROM departments d JOIN faculties f ON f.id=d.faculty_id WHERE f.university_id=$2) OR rm.course_id IN (SELECT co.id FROM courses co JOIN departments d ON d.id=co.department_id JOIN faculties f ON f.id=d.faculty_id WHERE f.university_id=$2)))`, c.UserID, id).Scan(&ok); err != nil {
		if s.Logger != nil {
			s.Logger.Warn("university scope query failed", "error", err, "userId", c.UserID, "universityId", id)
		}
		return false
	}
	return ok
}
func (s *Server) canAccessFaculty(ctx context.Context, c *auth.Claims, id uuid.UUID) bool {
	if hasRole(c, "super_admin") {
		return true
	}
	var ok bool
	_ = s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM faculties f JOIN resource_memberships rm ON (rm.university_id=f.university_id OR rm.faculty_id=f.id OR rm.department_id IN (SELECT id FROM departments WHERE faculty_id=f.id) OR rm.course_id IN (SELECT co.id FROM courses co JOIN departments d ON d.id=co.department_id WHERE d.faculty_id=f.id)) WHERE rm.user_id=$1 AND f.id=$2)`, c.UserID, id).Scan(&ok)
	return ok
}
func (s *Server) canAccessDepartment(ctx context.Context, c *auth.Claims, id uuid.UUID) bool {
	if hasRole(c, "super_admin") {
		return true
	}
	var ok bool
	_ = s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM departments d JOIN faculties f ON f.id=d.faculty_id JOIN resource_memberships rm ON (rm.university_id=f.university_id OR rm.faculty_id=f.id OR rm.department_id=d.id OR rm.course_id IN (SELECT id FROM courses WHERE department_id=d.id)) WHERE rm.user_id=$1 AND d.id=$2)`, c.UserID, id).Scan(&ok)
	return ok
}

func (s *Server) facultiesForUniversity(ctx context.Context, c *auth.Claims, universityID uuid.UUID) []map[string]interface{} {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,name,code FROM faculties WHERE university_id=$1 ORDER BY name`, universityID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, name, code string
		if rows.Scan(&id, &name, &code) == nil && s.canAccessFaculty(ctx, c, uuid.MustParse(id)) {
			out = append(out, map[string]interface{}{"id": id, "name": name, "code": code, "departments": s.departmentsForFaculty(ctx, c, uuid.MustParse(id))})
		}
	}
	return out
}
func (s *Server) departmentsForFaculty(ctx context.Context, c *auth.Claims, facultyID uuid.UUID) []map[string]interface{} {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,name,code FROM departments WHERE faculty_id=$1 ORDER BY name`, facultyID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, name, code string
		if rows.Scan(&id, &name, &code) == nil && s.canAccessDepartment(ctx, c, uuid.MustParse(id)) {
			out = append(out, map[string]interface{}{"id": id, "name": name, "code": code, "courses": s.coursesForDepartment(ctx, c, uuid.MustParse(id))})
		}
	}
	return out
}
func (s *Server) coursesForDepartment(ctx context.Context, c *auth.Claims, departmentID uuid.UUID) []map[string]interface{} {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,code,title,credits FROM courses WHERE department_id=$1 ORDER BY code`, departmentID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, code, title string
		var credits int
		if rows.Scan(&id, &code, &title, &credits) == nil {
			var allowed bool
			if hasRole(c, "super_admin") {
				allowed = true
			} else {
				_ = s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM resource_memberships rm WHERE rm.user_id=$1 AND (rm.course_id=$2 OR rm.department_id=$3 OR rm.faculty_id=(SELECT faculty_id FROM departments WHERE id=$3) OR rm.university_id=(SELECT f.university_id FROM departments d JOIN faculties f ON f.id=d.faculty_id WHERE d.id=$3))`, c.UserID, id, departmentID).Scan(&allowed)
			}
			if allowed {
				out = append(out, map[string]interface{}{"id": id, "code": code, "title": title, "credits": credits, "instructors": s.courseUsers(ctx, id, "instructor"), "students": s.courseUsers(ctx, id, "student")})
			}
		}
	}
	return out
}
func (s *Server) courseUsers(ctx context.Context, courseID, role string) []map[string]interface{} {
	query := `SELECT u.id,u.display_name,u.email FROM course_instructors ci JOIN users u ON u.id=ci.instructor_id WHERE ci.course_id=$1`
	if role == "student" {
		query = `SELECT u.id,u.display_name,u.email FROM course_students cs JOIN users u ON u.id=cs.student_id WHERE cs.course_id=$1`
	}
	rows, err := s.DB.QueryContext(ctx, query, courseID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, name, email string
		if rows.Scan(&id, &name, &email) == nil {
			out = append(out, map[string]interface{}{"id": id, "name": name, "email": email})
		}
	}
	return out
}

func (s *Server) organizationAssignments(w http.ResponseWriter, r *http.Request) {
	c := claimsOf(r)
	if c == nil || (!hasRole(c, "super_admin") && !hasRole(c, "university_admin")) {
		write(w, 403, map[string]string{"error": "organization administration required"})
		return
	}
	var in assignmentRequest
	if !decode(w, r, &in) {
		return
	}
	if in.UserID == "" || in.Role != "instructor" && in.Role != "proctor" && in.Role != "university_admin" {
		write(w, 400, map[string]string{"error": "userId and valid role are required"})
		return
	}
	userID, err := uuid.Parse(in.UserID)
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid userId"})
		return
	}
	uni, err := parseOptionalUUID(in.UniversityID)
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid universityId"})
		return
	}
	fac, err := parseOptionalUUID(in.FacultyID)
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid facultyId"})
		return
	}
	dept, err := parseOptionalUUID(in.DepartmentID)
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid departmentId"})
		return
	}
	course, err := parseOptionalUUID(in.CourseID)
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid courseId"})
		return
	}
	_, err = s.DB.ExecContext(r.Context(), `INSERT INTO resource_memberships(user_id,role_slug,university_id,faculty_id,department_id,course_id,created_by) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`, userID, in.Role, uni, fac, dept, course, c.UserID)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not assign resource scope"})
		return
	}
	write(w, 201, map[string]string{"status": "assigned"})
}

func (s *Server) enrollCourseStudent(w http.ResponseWriter, r *http.Request) {
	c := claimsOf(r)
	courseID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		write(w, 400, map[string]string{"error": "invalid course id"})
		return
	}
	if !s.canManageCourse(r.Context(), c, courseID) {
		write(w, 403, map[string]string{"error": "course scope denied"})
		return
	}
	var in courseEnrollmentRequest
	if !decode(w, r, &in) {
		return
	}
	studentID, err := uuid.Parse(in.UserID)
	if err != nil || !s.userHasRole(r.Context(), in.UserID, "student") {
		write(w, 400, map[string]string{"error": "valid student userId is required"})
		return
	}
	_, err = s.DB.ExecContext(r.Context(), `INSERT INTO course_students(course_id,student_id,assigned_by) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, courseID, studentID, c.UserID)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not enroll student"})
		return
	}
	write(w, 201, map[string]string{"status": "enrolled"})
}
func (s *Server) assignCourseInstructor(w http.ResponseWriter, r *http.Request) {
	c := claimsOf(r)
	courseID, err := uuid.Parse(r.PathValue("id"))
	if err != nil || !s.canManageCourse(r.Context(), c, courseID) {
		write(w, 403, map[string]string{"error": "course scope denied"})
		return
	}
	var in courseEnrollmentRequest
	if !decode(w, r, &in) {
		return
	}
	instructorID, err := uuid.Parse(in.UserID)
	if err != nil || !s.userHasRole(r.Context(), in.UserID, "instructor") {
		write(w, 400, map[string]string{"error": "valid instructor userId is required"})
		return
	}
	_, err = s.DB.ExecContext(r.Context(), `INSERT INTO course_instructors(course_id,instructor_id,assigned_by) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, courseID, instructorID, c.UserID)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not assign instructor"})
		return
	}
	write(w, 201, map[string]string{"status": "assigned"})
}
func (s *Server) assignExamProctor(w http.ResponseWriter, r *http.Request) {
	c := claimsOf(r)
	examID, err := uuid.Parse(r.PathValue("id"))
	if err != nil || !s.canManageExam(r.Context(), c, examID) {
		write(w, 403, map[string]string{"error": "exam scope denied"})
		return
	}
	var in examProctorRequest
	if !decode(w, r, &in) {
		return
	}
	proctorID, err := uuid.Parse(in.UserID)
	if err != nil || !s.userHasRole(r.Context(), in.UserID, "proctor") {
		write(w, 400, map[string]string{"error": "valid proctor userId is required"})
		return
	}
	_, err = s.DB.ExecContext(r.Context(), `INSERT INTO exam_proctors(exam_id,proctor_id,assigned_by) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, examID, proctorID, c.UserID)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not assign proctor"})
		return
	}
	write(w, 201, map[string]string{"status": "assigned"})
}

func (s *Server) organizationOverview(w http.ResponseWriter, r *http.Request) {
	c := claimsOf(r)
	if c == nil {
		write(w, 401, map[string]string{"error": "authentication required"})
		return
	}
	var universities, faculties, departments, courses, students, instructors, proctors, exams int
	_ = s.DB.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM universities`).Scan(&universities)
	_ = s.DB.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM faculties`).Scan(&faculties)
	_ = s.DB.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM departments`).Scan(&departments)
	_ = s.DB.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM courses`).Scan(&courses)
	_ = s.DB.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM course_students cs WHERE EXISTS(SELECT 1 FROM resource_memberships rm WHERE rm.user_id=$1 OR $2)`, c.UserID, hasRole(c, "super_admin")).Scan(&students)
	_ = s.DB.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM course_instructors ci WHERE EXISTS(SELECT 1 FROM resource_memberships rm WHERE rm.user_id=$1 OR $2)`, c.UserID, hasRole(c, "super_admin")).Scan(&instructors)
	_ = s.DB.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM exam_proctors ep WHERE ep.proctor_id=$1 OR $2`, c.UserID, hasRole(c, "super_admin")).Scan(&proctors)
	_ = s.DB.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM exams e JOIN courses co ON co.id=e.course_id WHERE e.deleted_at IS NULL AND ($2 OR EXISTS(SELECT 1 FROM resource_memberships rm WHERE rm.user_id=$1 AND (rm.course_id=co.id OR rm.department_id=co.department_id)))`, c.UserID, hasRole(c, "super_admin")).Scan(&exams)
	write(w, 200, map[string]interface{}{"universities": universities, "faculties": faculties, "departments": departments, "courses": courses, "students": students, "instructors": instructors, "proctors": proctors, "exams": exams})
}

func marshal(v interface{}) []byte { b, _ := json.Marshal(v); return b }
