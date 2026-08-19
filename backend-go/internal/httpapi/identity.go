package httpapi

import (
	"context"
	"database/sql"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var studentCodePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{2,79}$`)

type studentCodeRequest struct {
	StudentCode string `json:"studentCode"`
}

type studentIdentityResponse struct {
	StudentCode string `json:"studentCode"`
	Status      string `json:"status"`
	Verified    bool   `json:"verified"`
	Message     string `json:"message"`
}

type adminIdentityRow struct {
	StudentCode string    `json:"studentCode"`
	Status      string    `json:"status"`
	UserID      string    `json:"userId"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	CreatedAt   time.Time `json:"createdAt"`
}

type studentCandidate struct {
	UserID      string `json:"userId"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	StudentCode string `json:"studentCode"`
}

type bulkIdentityLink struct {
	StudentCode string `json:"studentCode"`
	UserID      string `json:"userId"`
}

type bulkIdentityRequest struct {
	Links []bulkIdentityLink `json:"links"`
}

type identityLinkError struct {
	status  int
	message string
}

func (e identityLinkError) Error() string { return e.message }

func validStudentCode(code string) bool {
	return studentCodePattern.MatchString(strings.TrimSpace(code))
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
	err := s.DB.QueryRowContext(r.Context(), `SELECT st.student_number,COALESCE(reg.status,'unregistered'),reg.user_id::text FROM students st LEFT JOIN student_identity_registry reg ON reg.student_number=st.student_number WHERE st.user_id=$1`, c.UserID).Scan(&code, &status, &registryUserID)
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
	if !validStudentCode(code) {
		write(w, http.StatusBadRequest, map[string]string{"error": "valid studentCode is required"})
		return
	}
	var status string
	var registryUserID *string
	err := s.DB.QueryRowContext(r.Context(), `SELECT COALESCE(reg.status,'unregistered'),reg.user_id::text FROM students st LEFT JOIN student_identity_registry reg ON reg.student_number=st.student_number WHERE st.user_id=$1 AND st.student_number=$2`, c.UserID, code).Scan(&status, &registryUserID)
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

func (s *Server) listStudentIdentities(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(), `SELECT reg.student_number,reg.status,COALESCE(reg.user_id::text,''),COALESCE(u.email,''),COALESCE(u.display_name,''),reg.created_at FROM student_identity_registry reg LEFT JOIN users u ON u.id=reg.user_id ORDER BY CASE reg.status WHEN 'pending' THEN 0 WHEN 'linked' THEN 1 ELSE 2 END,reg.student_number LIMIT 2000`)
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"error": "could not load student identities"})
		return
	}
	defer rows.Close()
	out := make([]adminIdentityRow, 0)
	for rows.Next() {
		var item adminIdentityRow
		if err := rows.Scan(&item.StudentCode, &item.Status, &item.UserID, &item.Email, &item.DisplayName, &item.CreatedAt); err == nil {
			out = append(out, item)
		}
	}
	write(w, http.StatusOK, out)
}

func (s *Server) listStudentCandidates(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(), `SELECT u.id::text,u.email,u.display_name,COALESCE(st.student_number,'') FROM users u JOIN user_roles ur ON ur.user_id=u.id JOIN roles ro ON ro.id=ur.role_id AND ro.slug='student' LEFT JOIN students st ON st.user_id=u.id WHERE u.is_active=true AND u.deleted_at IS NULL ORDER BY u.display_name,u.email`)
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"error": "could not load student candidates"})
		return
	}
	defer rows.Close()
	out := make([]studentCandidate, 0)
	for rows.Next() {
		var item studentCandidate
		if err := rows.Scan(&item.UserID, &item.Email, &item.DisplayName, &item.StudentCode); err == nil {
			out = append(out, item)
		}
	}
	write(w, http.StatusOK, out)
}

func (s *Server) linkIdentityTx(ctx context.Context, tx *sql.Tx, in bulkIdentityLink) identityLinkError {
	code := strings.TrimSpace(in.StudentCode)
	studentID, err := uuid.Parse(strings.TrimSpace(in.UserID))
	if !validStudentCode(code) || err != nil || !s.userHasRole(ctx, studentID.String(), "student") {
		return identityLinkError{status: http.StatusBadRequest, message: "valid studentCode and student userId are required"}
	}
	var conflict bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM student_identity_registry WHERE student_number=$1 AND user_id IS NOT NULL AND user_id<>$2) OR EXISTS(SELECT 1 FROM students WHERE student_number=$1 AND user_id<>$2)`, code, studentID).Scan(&conflict); err != nil {
		return identityLinkError{status: http.StatusInternalServerError, message: "could not validate identity conflict"}
	}
	if conflict {
		return identityLinkError{status: http.StatusConflict, message: "student code is already linked to another account"}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO students(user_id,student_number) VALUES($1,$2) ON CONFLICT(user_id) DO UPDATE SET student_number=EXCLUDED.student_number`, studentID, code); err != nil {
		return identityLinkError{status: http.StatusConflict, message: "could not attach student code to account"}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO student_identity_registry(student_number,user_id,status,source,verified_at,updated_at) VALUES($1,$2,'linked','admin-bulk-link',now(),now()) ON CONFLICT(student_number) DO UPDATE SET user_id=EXCLUDED.user_id,status='linked',source='admin-bulk-link',verified_at=now(),updated_at=now()`, code, studentID); err != nil {
		return identityLinkError{status: http.StatusConflict, message: "could not update identity registry"}
	}
	return identityLinkError{}
}

func (s *Server) linkStudentCode(w http.ResponseWriter, r *http.Request) {
	c := claimsOf(r)
	var in bulkIdentityLink
	if !decode(w, r, &in) {
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"error": "could not begin identity link"})
		return
	}
	defer tx.Rollback()
	if linkErr := s.linkIdentityTx(r.Context(), tx, in); linkErr.status != 0 {
		write(w, linkErr.status, map[string]string{"error": linkErr.message})
		return
	}
	if err = tx.Commit(); err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"error": "could not commit identity link"})
		return
	}
	s.audit(r.Context(), c.UserID, "STUDENT_IDENTITY_LINKED", "student_identity_registry", uuid.Nil, "SUCCESS", map[string]interface{}{"studentCode": strings.TrimSpace(in.StudentCode), "userId": strings.TrimSpace(in.UserID)})
	write(w, http.StatusCreated, map[string]string{"status": "linked", "studentCode": strings.TrimSpace(in.StudentCode), "userId": strings.TrimSpace(in.UserID)})
}

func (s *Server) bulkLinkStudentCodes(w http.ResponseWriter, r *http.Request) {
	c := claimsOf(r)
	var in bulkIdentityRequest
	if !decode(w, r, &in) {
		return
	}
	if len(in.Links) == 0 || len(in.Links) > 2000 {
		write(w, http.StatusBadRequest, map[string]string{"error": "links must contain between 1 and 2000 items"})
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"error": "could not begin bulk identity link"})
		return
	}
	defer tx.Rollback()
	for i, link := range in.Links {
		if linkErr := s.linkIdentityTx(r.Context(), tx, link); linkErr.status != 0 {
			write(w, linkErr.status, map[string]interface{}{"error": linkErr.message, "row": i + 1})
			return
		}
	}
	if err = tx.Commit(); err != nil {
		write(w, http.StatusInternalServerError, map[string]string{"error": "could not commit bulk identity link"})
		return
	}
	s.audit(r.Context(), c.UserID, "STUDENT_IDENTITY_BULK_LINKED", "student_identity_registry", uuid.Nil, "SUCCESS", map[string]interface{}{"count": len(in.Links)})
	write(w, http.StatusCreated, map[string]interface{}{"status": "linked", "linked": len(in.Links)})
}
