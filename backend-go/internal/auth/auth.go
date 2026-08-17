package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	UserID string   `json:"sub"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

type Service struct {
	Secret    []byte
	AccessTTL time.Duration
}

func HashPassword(password string) (string, error) {
	if len(password) < 12 {
		return "", errors.New("password must contain at least 12 characters")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
func (s Service) Issue(userID string, roles []string) (string, error) {
	now := time.Now()
	c := Claims{UserID: userID, Roles: roles, RegisteredClaims: jwt.RegisteredClaims{Issuer: "hnu-test", IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(s.AccessTTL))}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(s.Secret)
}
func (s Service) Parse(token string) (*Claims, error) {
	token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return s.Secret, nil
	})
	if err != nil || !parsed.Valid {
		return nil, errors.New("invalid access token")
	}
	c, ok := parsed.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return c, nil
}
func HasRole(roles []string, expected ...string) bool {
	for _, have := range roles {
		for _, want := range expected {
			if have == want {
				return true
			}
		}
	}
	return false
}
