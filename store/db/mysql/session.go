package mysql

import (
	"context"
	"database/sql"

	sessionUtils "github.com/sagarsuperuser/userprofile/internal/session"
	"github.com/sagarsuperuser/userprofile/store"
)

func (d *DB) CreateSession(ctx context.Context, userID int64, hash [32]byte) (*store.SessionInfo, error) {
	now := d.now()
	expiresAt := now.Add(sessionUtils.SessionDuration)

	// insert sesssion
	res, err := d.db.ExecContext(ctx, `
			INSERT INTO sessions (user_id, token_hash, expires_at, created_at)
			VALUES (?, ?, ?, ?)
	`, userID, hash[:], expiresAt, now)

	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &store.SessionInfo{
		ID:        id,
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
		RevokedAt: nil,
		IsActive:  true,
	}, nil
}

func (d *DB) GetActiveSessionByHash(ctx context.Context, hash [32]byte) (*store.SessionInfo, error) {
	now := d.now()
	var s store.SessionInfo
	err := d.db.QueryRowContext(ctx, `
		SELECT id, user_id, expires_at, created_at, revoked_at
		FROM sessions
		WHERE token_hash = ?
		AND revoked_at IS NULL
		AND expires_at > ?
		LIMIT 1
	`, hash[:], now).Scan(
		&s.ID,
		&s.UserID,
		&s.ExpiresAt,
		&s.CreatedAt,
		&s.RevokedAt,
	)

	if err == sql.ErrNoRows {
		return nil, sessionUtils.ErrSesssionExpired
	}
	if err != nil {
		return nil, err
	}

	s.IsActive = true
	return &s, nil
}

func (d *DB) RevokeSession(ctx context.Context, hash [32]byte) (bool, error) {
	now := d.now()
	res, err := d.db.ExecContext(ctx, `
        UPDATE sessions
        SET revoked_at = ?
        WHERE token_hash = ? AND revoked_at IS NULL
    `, now, hash[:])
	if err != nil {
		return false, err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil // false => already revoked or not found
}
