package auth

import (
	"testing"
	"time"
)

func TestPasswordHashing(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil || !CheckPassword(hash, "correct horse battery staple") || CheckPassword(hash, "wrong password") {
		t.Fatal("password hashing contract failed")
	}
}
func TestShortPasswordRejected(t *testing.T) {
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("short password accepted")
	}
}
func TestJWTClaims(t *testing.T) {
	s := Service{Secret: []byte("a sufficiently long local secret"), AccessTTL: 15 * time.Minute}
	token, err := s.Issue("user-1", []string{"instructor"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.Parse(token)
	if err != nil || c.UserID != "user-1" || !HasRole(c.Roles, "instructor") {
		t.Fatal("claims were not preserved")
	}
}
