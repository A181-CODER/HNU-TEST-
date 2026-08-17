package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/A181-CODER/HNU-TEST-/backend-go/internal/auth"
	"github.com/A181-CODER/HNU-TEST-/backend-go/internal/httpapi"
	_ "github.com/lib/pq"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	port := getenv("APP_PORT", "8080")
	secret := getenv("JWT_SECRET", "local-development-secret-change-me-please")
	ttl, _ := time.ParseDuration(getenv("ACCESS_TOKEN_MINUTES", "15") + "m")
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	var db *sql.DB
	if url := os.Getenv("DATABASE_URL"); url != "" {
		var err error
		db, err = sql.Open("postgres", url)
		if err != nil {
			logger.Error("database open failed", "error", err)
			os.Exit(1)
		}
	}
	s := &httpapi.Server{DB: db, Auth: auth.Service{Secret: []byte(secret), AccessTTL: ttl}, Logger: logger, CORS: getenv("CORS_ORIGINS", "http://localhost:5173")}
	logger.Info("starting HNU TEST API", "port", port)
	if err := http.ListenAndServe(":"+port, s.Routes()); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
func getenv(k, d string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return d
	}
	return v
}

var _ = strconv.Itoa
