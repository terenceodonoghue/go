package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.local/services/auth/internal/db"
	"go.local/services/auth/internal/handler"
	"go.local/services/auth/internal/store"
)

func main() {
	ctx := context.Background()

	dbURL := requiredEnv("DATABASE_URL")
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	redisAddr := requiredEnv("REDIS_ADDR")
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer rdb.Close()

	rpID := requiredEnv("RP_ID")
	rpOrigin := requiredEnv("RP_ORIGIN")
	wconfig := &webauthn.Config{
		RPID:          rpID,
		RPDisplayName: "Auth",
		RPOrigins:     []string{rpOrigin},
	}

	webAuthn, err := webauthn.New(wconfig)
	if err != nil {
		log.Fatalf("Failed to initialise WebAuthn: %v", err)
	}

	h := &handler.Handler{
		WebAuthn:             webAuthn,
		Queries:              db.New(pool),
		Store:                store.NewRedisStore(rdb),
		SecureCookie:         strings.HasPrefix(rpOrigin, "https://"),
		LogVerificationCodes: os.Getenv("LOG_VERIFICATION_CODES") == "true",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/register/begin", h.BeginRegistration)
	mux.HandleFunc("POST /api/register/verify", h.VerifyAndBeginPasskey)
	mux.HandleFunc("POST /api/register/finish", h.FinishRegistration)
	mux.HandleFunc("POST /api/login/begin", h.BeginLogin)
	mux.HandleFunc("POST /api/login/finish", h.FinishLogin)

	addr := ":8081"
	if v := os.Getenv("ADDR"); v != "" {
		addr = v
	}

	log.Println("Configuration:")
	log.Printf("  ADDR                   = %s", addr)
	log.Printf("  DATABASE_URL           = %s", dbURL)
	log.Printf("  LOG_VERIFICATION_CODES = %v", h.LogVerificationCodes)
	log.Printf("  REDIS_ADDR             = %s", redisAddr)
	log.Printf("  RP_ID                  = %s", rpID)
	log.Printf("  RP_ORIGIN              = %s", rpOrigin)
	log.Println()

	log.Printf("Auth server listening on %s", addr)
	if err := http.ListenAndServe(addr, handler.CORS([]string{rpOrigin}, mux)); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func requiredEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s is required", key)
	}
	return v
}
