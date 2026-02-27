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
WebAuthn ceremony sessions hold challenge data between the begin and finish steps
of a login ceremony. Sessions expire after 5 minutes.
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
Registration sessions carry the display name and WebAuthn ceremony data between
the begin and finish steps of a registration. Sessions expire after 5 minutes.
*/

type RegistrationSession struct {
	DisplayName string               `json:"display_name"`
	WebAuthn    *webauthn.SessionData `json:"webauthn"`
}

func (s *RedisStore) SaveRegistrationSession(ctx context.Context, sessionID string, data *RegistrationSession) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, registrationKey(sessionID), b, 5*time.Minute).Err()
}

func (s *RedisStore) GetRegistrationSession(ctx context.Context, sessionID string) (*RegistrationSession, error) {
	b, err := s.client.Get(ctx, registrationKey(sessionID)).Bytes()
	if err != nil {
		return nil, err
	}
	var data RegistrationSession
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (s *RedisStore) DeleteRegistrationSession(ctx context.Context, sessionID string) error {
	return s.client.Del(ctx, registrationKey(sessionID)).Err()
}

/*
Auth sessions persist user identity after a successful registration or login.
Sessions have a 15-minute sliding TTL that refreshes on each access.
*/

type AuthSession struct {
	DisplayName string `json:"display_name"`
}

func (s *RedisStore) SaveAuthSession(ctx context.Context, token string, session *AuthSession) error {
	b, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, authSessionKey(token), b, 15*time.Minute).Err()
}

func (s *RedisStore) GetAuthSession(ctx context.Context, token string) (*AuthSession, error) {
	key := authSessionKey(token)
	b, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	s.client.Expire(ctx, key, 15*time.Minute)
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

func registrationKey(sessionID string) string {
	return fmt.Sprintf("registration:session:%s", sessionID)
}

func sessionKey(sessionID string) string {
	return fmt.Sprintf("webauthn:session:%s", sessionID)
}
