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
	"go.local/pkg/env"
	"go.local/services/auth-api/internal/db"
	"go.local/services/auth-api/internal/handler"
	"go.local/services/auth-api/internal/store"
)

func main() {
	ctx := context.Background()

	dbURL := env.Required("DATABASE_URL")
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	redisAddr := env.Required("REDIS_ADDR")
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer rdb.Close()

	rpID := env.Required("RP_ID")
	rpOrigin := env.Required("RP_ORIGIN")
	wconfig := &webauthn.Config{
		RPID:          rpID,
		RPDisplayName: "Auth",
		RPOrigins:     []string{rpOrigin},
	}

	webAuthn, err := webauthn.New(wconfig)
	if err != nil {
		log.Fatalf("Failed to initialise WebAuthn: %v", err)
	}

	loginURL := os.Getenv("LOGIN_URL")
	if loginURL == "" {
		log.Println("WARNING: LOGIN_URL is not set; /api/verify will not redirect unauthenticated web requests")
	}

	h := &handler.Handler{
		WebAuthn:             webAuthn,
		Queries:              db.New(pool),
		Store:                store.NewRedisStore(rdb),
		SecureCookie:         strings.HasPrefix(rpOrigin, "https://"),
		LogVerificationCodes: os.Getenv("LOG_VERIFICATION_CODES") == "true",
		LoginURL:             loginURL,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/register/begin", h.BeginRegistration)
	mux.HandleFunc("POST /api/register/verify", h.VerifyAndBeginPasskey)
	mux.HandleFunc("POST /api/register/finish", h.FinishRegistration)
	mux.HandleFunc("POST /api/login/begin", h.BeginLogin)
	mux.HandleFunc("POST /api/login/finish", h.FinishLogin)
	mux.HandleFunc("GET /api/verify", h.VerifySession)
	mux.HandleFunc("POST /api/logout", h.Logout)

	addr := ":8081"
	if v := os.Getenv("ADDR"); v != "" {
		addr = v
	}

	log.Println("Configuration:")
	log.Printf("  ADDR                   = %s", addr)
	log.Printf("  DATABASE_URL           = %s", dbURL)
	log.Printf("  LOG_VERIFICATION_CODES = %v", h.LogVerificationCodes)
	log.Printf("  LOGIN_URL              = %s", loginURL)
	log.Printf("  REDIS_ADDR             = %s", redisAddr)
	log.Printf("  RP_ID                  = %s", rpID)
	log.Printf("  RP_ORIGIN              = %s", rpOrigin)
	log.Println()

	log.Printf("Auth server listening on %s", addr)
	if err := http.ListenAndServe(addr, handler.CORS([]string{rpOrigin}, mux)); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
