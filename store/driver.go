package store

import (
	"context"
	"database/sql"
)

// Driver is an interface for store driver.
// It contains all methods that store database driver should implement.
type Driver interface {
	GetDB() *sql.DB
	Close() error

	// users model related methods.
	CreateUser(ctx context.Context, create *CreateUser) (*UserInfo, error)
	ListUsers(ctx context.Context, find *FindUser) ([]*UserInfo, error)
	UpdateUser(ctx context.Context, update *UpdateUser) (*UserInfo, error)
	DeleteUser(ctx context.Context, delete *DeleteUser) (bool, error)

	// sessions model related methods
	CreateSession(ctx context.Context, userID int64, hash [32]byte) (*SessionInfo, error)
	GetActiveSessionByHash(ctx context.Context, hash [32]byte) (*SessionInfo, error)
	RevokeSession(ctx context.Context, hash [32]byte) (bool, error)
}
