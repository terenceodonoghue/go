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

/*
Email verification codes confirm ownership of an email address during registration.
Codes expire after 10 minutes.
*/

func (s *RedisStore) SaveVerification(ctx context.Context, email, code string) error {
	return s.client.Set(ctx, verifyKey(email), code, 10*time.Minute).Err()
}

func (s *RedisStore) GetVerification(ctx context.Context, email string) (string, error) {
	return s.client.Get(ctx, verifyKey(email)).Result()
}

func (s *RedisStore) DeleteVerification(ctx context.Context, email string) error {
	return s.client.Del(ctx, verifyKey(email)).Err()
}

/*
WebAuthn ceremony sessions hold challenge data between the begin and finish steps
of registration or login. Sessions expire after 5 minutes.
*/

func (s *RedisStore) SaveWebAuthnSession(ctx context.Context, sessionID string, data *webauthn.SessionData) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, sessionKey(sessionID), b, 5*time.Minute).Err()
}

func (s *RedisStore) GetWebAuthnSession(ctx context.Context, sessionID string) (*webauthn.SessionData, error) {
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

func (s *RedisStore) DeleteWebAuthnSession(ctx context.Context, sessionID string) error {
	return s.client.Del(ctx, sessionKey(sessionID)).Err()
}

/*
Auth sessions persist user identity after a successful registration or login.
Sessions have a 24-hour sliding TTL that refreshes on each access.
*/

type AuthSession struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

func (s *RedisStore) SaveAuthSession(ctx context.Context, token string, session *AuthSession) error {
	b, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, authSessionKey(token), b, 24*time.Hour).Err()
}

func (s *RedisStore) GetAuthSession(ctx context.Context, token string) (*AuthSession, error) {
	key := authSessionKey(token)
	b, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	s.client.Expire(ctx, key, 24*time.Hour)
	var session AuthSession
	if err := json.Unmarshal(b, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *RedisStore) DeleteAuthSession(ctx context.Context, token string) error {
	return s.client.Del(ctx, authSessionKey(token)).Err()
}

func authSessionKey(token string) string {
	return fmt.Sprintf("auth:session:%s", token)
}

func verifyKey(email string) string {
	return fmt.Sprintf("verify:%s", email)
}

func sessionKey(sessionID string) string {
	return fmt.Sprintf("webauthn:session:%s", sessionID)
}
