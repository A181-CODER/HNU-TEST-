package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type studentCodeRequest struct {
	StudentCode string `json:"studentCode"`
}

type studentIdentityResponse struct {
	StudentCode string `json:"studentCode"`
	Status      string `json:"status"`
	Verified    bool   `json:"verified"`
	Message     string `json:"message"`
}

func (s *Server) studentIdentityLinked(ctx context.Context, userID string) bool {
	var linked bool
	err := s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM student_identity_registry reg JOIN students st ON st.student_number=reg.student_number WHERE reg.user_id=$1 AND st.user_id=$1 AND reg.status='linked')`, userID).Scan(&linked)
	return err == nil && linked
}

func (s *Server) studentIdentity(w http.ResponseWriter, r *http.Request) {
	c := claimsOf(r)
	var code, status string
	var registryUserID *string
	err := s.DB.QueryRowContext(r.Context(), `
		SELECT st.student_number, COALESCE(reg.status,'unregistered'), reg.user_id::text
		FROM students st
		LEFT JOIN student_identity_registry reg ON reg.student_number=st.student_number
		WHERE st.user_id=$1`, c.UserID).Scan(&code, &status, &registryUserID)
	if err != nil {
		write(w, http.StatusNotFound, map[string]string{"error": "student identity is not registered"})
		return
	}
	verified := status == "linked" && registryUserID != nil && *registryUserID == c.UserID
	message := "student code is registered but requires controlled linking"
	if verified {
		message = "student code is linked to this authenticated student account"
	}
	write(w, http.StatusOK, studentIdentityResponse{StudentCode: code, Status: status, Verified: verified, Message: message})
}

func (s *Server) verifyStudentCode(w http.ResponseWriter, r *http.Request) {
	c := claimsOf(r)
	var in studentCodeRequest
	if !decode(w, r, &in) {
		return
	}
	code := strings.TrimSpace(in.StudentCode)
	if code == "" || len(code) > 80 {
		write(w, http.StatusBadRequest, map[string]string{"error": "studentCode is required"})
		return
	}
	var status string
	var registryUserID *string
	err := s.DB.QueryRowContext(r.Context(), `
		SELECT COALESCE(reg.status,'unregistered'), reg.user_id::text
		FROM students st
		LEFT JOIN student_identity_registry reg ON reg.student_number=st.student_number
		WHERE st.user_id=$1 AND st.student_number=$2`, c.UserID, code).Scan(&status, &registryUserID)
	if err != nil || status == "disabled" {
		write(w, http.StatusForbidden, studentIdentityResponse{StudentCode: code, Status: "rejected", Verified: false, Message: "student code does not match the authenticated account"})
		return
	}
	verified := status == "linked" && registryUserID != nil && *registryUserID == c.UserID
	if !verified {
		write(w, http.StatusConflict, studentIdentityResponse{StudentCode: code, Status: status, Verified: false, Message: "student code matches the account but is not linked in the controlled registry"})
		return
	}
	write(w, http.StatusOK, studentIdentityResponse{StudentCode: code, Status: status, Verified: true, Message: "student identity verified"})
}

func (s *Server) linkStudentCode(w http.ResponseWriter, r *http.Request) {
	c := claimsOf(r)
	if !hasRole(c, "super_admin") && !hasRole(c, "university_admin") {
		write(w, http.StatusForbidden, map[string]string{"error": "only university administrators can link student identities"})
		return
	}
	var in struct {
		StudentCode string `json:"studentCode"`
		UserID      string `json:"userId"`
	}
	if !decode(w, r, &in) {
		return
	}
	code := strings.TrimSpace(in.StudentCode)
	studentID, err := uuid.Parse(strings.TrimSpace(in.UserID))
	if code == "" || err != nil || !s.userHasRole(r.Context(), studentID.String(), "student") {
		write(w, http.StatusBadRequest, map[string]string{"error": "valid studentCode and student userId are required"})
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"error": "could not begin identity link"})
		return
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(r.Context(), `INSERT INTO students(user_id,student_number) VALUES($1,$2) ON CONFLICT(user_id) DO UPDATE SET student_number=EXCLUDED.student_number`, studentID, code); err != nil {
		write(w, http.StatusConflict, map[string]string{"error": "student code is already assigned or could not be linked"})
		return
	}
	if _, err = tx.ExecContext(r.Context(), `INSERT INTO student_identity_registry(student_number,user_id,status,source,verified_at,updated_at) VALUES($1,$2,'linked','admin-link',now(),now()) ON CONFLICT(student_number) DO UPDATE SET user_id=EXCLUDED.user_id,status='linked',source='admin-link',verified_at=now(),updated_at=now()`, code, studentID); err != nil {
		write(w, http.StatusConflict, map[string]string{"error": "could not update identity registry"})
		return
	}
	if err = tx.Commit(); err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"error": "could not commit identity link"})
		return
	}
	s.audit(context.Background(), c.UserID, "STUDENT_IDENTITY_LINKED", "student_identity_registry", studentID, "SUCCESS", map[string]interface{}{"studentCode": code})
	write(w, http.StatusCreated, map[string]string{"status": "linked", "studentCode": code, "userId": studentID.String()})
}
