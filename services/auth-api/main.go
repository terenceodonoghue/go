package main

import (
	"context"
	_ "embed"
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

//go:embed sql/schema.sql
var schema string

func main() {
	ctx := context.Background()

	dbURL := env.Required("DATABASE_URL")
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, schema); err != nil {
		log.Fatalf("Failed to apply database schema: %v", err)
	}
	log.Println("Database schema applied")

	redisAddr := env.Required("REDIS_ADDR")
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer rdb.Close()

	rpID := env.Required("RP_ID")
	rpOrigins := strings.Split(env.Required("RP_ORIGINS"), ",")
	wconfig := &webauthn.Config{
		RPID:          rpID,
		RPDisplayName: "Auth",
		RPOrigins:     rpOrigins,
	}

	webAuthn, err := webauthn.New(wconfig)
	if err != nil {
		log.Fatalf("Failed to initialise WebAuthn: %v", err)
	}

	rpOrigin := rpOrigins[0]
	h := &handler.Handler{
		WebAuthn:     webAuthn,
		Queries:      db.New(pool),
		Store:        store.NewRedisStore(rdb),
		SecureCookie: strings.HasPrefix(rpOrigin, "https://"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/register/begin", h.BeginPasskeyRegistration)
	mux.HandleFunc("POST /api/register/finish", h.FinishRegistration)
	mux.HandleFunc("POST /api/login/begin", h.BeginLogin)
	mux.HandleFunc("POST /api/login/finish", h.FinishLogin)
	mux.HandleFunc("GET /api/verify", h.VerifySession)
	mux.HandleFunc("GET /api/network", h.GetNetworkContext)
	mux.HandleFunc("POST /api/logout", h.Logout)

	addr := ":8081"
	if v := os.Getenv("ADDR"); v != "" {
		addr = v
	}

	log.Println("Configuration:")
	log.Printf("  ADDR        = %s", addr)
	log.Printf("  DATABASE_URL = %s", dbURL)
	log.Printf("  REDIS_ADDR  = %s", redisAddr)
	log.Printf("  RP_ID       = %s", rpID)
	log.Printf("  RP_ORIGINS  = %s", strings.Join(rpOrigins, ", "))
	log.Println()

	log.Printf("Auth server listening on %s", addr)
	if err := http.ListenAndServe(addr, handler.CORS(rpOrigins, mux)); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
