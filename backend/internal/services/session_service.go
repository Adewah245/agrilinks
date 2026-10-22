package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

type sessionRepository interface {
	Create(context.Context, string, int, time.Time) error
	UserID(context.Context, string) (int, error)
	Delete(context.Context, string) error
	RevokeAll(context.Context, int) error
}

type SessionService struct {
	repo sessionRepository
	ttl  time.Duration
}

func NewSessionService(repo sessionRepository, ttl time.Duration) *SessionService {
	return &SessionService{repo: repo, ttl: ttl}
}

func (s *SessionService) TTL() time.Duration { return s.ttl }

func (s *SessionService) Create(ctx context.Context, userID int) (string, error) {
	if userID <= 0 || s.ttl <= 0 {
		return "", errors.New("invalid session settings")
	}
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	token := hex.EncodeToString(bytes[:])
	if err := s.repo.Create(ctx, sessionHash(token), userID, time.Now().Add(s.ttl)); err != nil {
		return "", err
	}
	return token, nil
}

func (s *SessionService) UserID(ctx context.Context, token string) (int, error) {
	if len(token) != 64 {
		return 0, nil
	}
	return s.repo.UserID(ctx, sessionHash(token))
}

func (s *SessionService) Delete(ctx context.Context, token string) error {
	return s.repo.Delete(ctx, sessionHash(token))
}

func (s *SessionService) RevokeAll(ctx context.Context, userID int) error {
	return s.repo.RevokeAll(ctx, userID)
}

func sessionHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
