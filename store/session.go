package store

import (
	"context"
	"crypto/sha256"
	"time"

	sessionUtils "github.com/sagarsuperuser/userprofile/internal/session"
)

type SessionInfo struct {
	ID     int64
	UserID int64
	// SHA-256 hash of [Token]
	TokenHash [32]byte
	ExpiresAt time.Time
	RevokedAt *time.Time // nil - not revoked
	CreatedAt time.Time
	IsActive  bool
}

type CreateSessionResult struct {
	// Raw token to be sent to the browser as a cookie value
	Token string
	// DB session metadata
	Session SessionInfo
}

// create new session
func (s *Store) CreateSession(ctx context.Context, userID int64) (*CreateSessionResult, error) {
	// generate token
	token := sessionUtils.GenerateSessionID()
	hash := sha256.Sum256([]byte(token))

	session, err := s.driver.CreateSession(ctx, userID, hash)
	if err != nil {
		return nil, err
	}
	// Cache the SHA-256 hash of the session token
	s.sessionCache.Store(hash, session)

	res := CreateSessionResult{}
	res.Token = token
	res.Session = *session
	return &res, nil
}

// get user active session, used by auth middleware
func (s *Store) GetActiveSessionByToken(ctx context.Context, token string) (*SessionInfo, error) {
	hash := sha256.Sum256([]byte(token))
	// check first in cache
	if v, ok := s.sessionCache.Load(hash); ok {
		sInfo := v.(SessionInfo)
		if sInfo.ExpiresAt.After(s.now()) {
			return nil, sessionUtils.ErrSesssionExpired
		}
		return &sInfo, nil
	}
	sInfo, err := s.driver.GetActiveSessionByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	// cache it for next requests
	s.sessionCache.Store(hash, sInfo)
	return sInfo, nil
}

// expire session, used by handlers
func (s *Store) RevokeSession(ctx context.Context, session SessionInfo) (bool, error) {
	// clear cache first
	s.sessionCache.Delete(session.TokenHash)

	// delete from database
	return s.driver.RevokeSession(ctx, session.TokenHash)
}
