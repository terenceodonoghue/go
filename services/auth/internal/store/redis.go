package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// SaveVerification stores a verification code for the given email with a 10-minute TTL.
func (s *RedisStore) SaveVerification(ctx context.Context, email, code string) error {
	return s.client.Set(ctx, verifyKey(email), code, 10*time.Minute).Err()
}

// GetVerification retrieves the verification code for the given email.
func (s *RedisStore) GetVerification(ctx context.Context, email string) (string, error) {
	return s.client.Get(ctx, verifyKey(email)).Result()
}

// DeleteVerification removes the verification code for the given email.
func (s *RedisStore) DeleteVerification(ctx context.Context, email string) error {
	return s.client.Del(ctx, verifyKey(email)).Err()
}

// SaveSession stores WebAuthn session data with a 5-minute TTL.
func (s *RedisStore) SaveSession(ctx context.Context, sessionID string, data *webauthn.SessionData) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, sessionKey(sessionID), b, 5*time.Minute).Err()
}

// GetSession retrieves WebAuthn session data by session ID.
func (s *RedisStore) GetSession(ctx context.Context, sessionID string) (*webauthn.SessionData, error) {
	b, err := s.client.Get(ctx, sessionKey(sessionID)).Bytes()
	if err != nil {
		return nil, err
	}
	var data webauthn.SessionData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteSession removes WebAuthn session data by session ID.
func (s *RedisStore) DeleteSession(ctx context.Context, sessionID string) error {
	return s.client.Del(ctx, sessionKey(sessionID)).Err()
}

func verifyKey(email string) string {
	return fmt.Sprintf("verify:%s", email)
}

func sessionKey(sessionID string) string {
	return fmt.Sprintf("webauthn:session:%s", sessionID)
}
