package store

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrUserAlreadyExists     = errors.New("user already exists")
	ErrUserNotFound          = errors.New("user not found")
	ErrEmailUpdateNotAllowed = errors.New("email update is not allowed")
)

// Role is the type of a role.
type Role string

const (
	// RoleAdmin is the ADMIN role.
	RoleAdmin Role = "admin"
	// RoleUser is the USER role.
	RoleUser Role = "user"
)

func (e Role) String() string {
	switch e {
	case RoleAdmin:
		return "admin"
	case RoleUser:
		return "user"
	default:
		return "unknown"
	}
}

type UserStatus string

const (
	StatusActive   UserStatus = "active"
	StatusDisabled UserStatus = "disabled"
)

type Provider string

const (
	ProviderGoogle Provider = "google"
	ProviderLocal  Provider = "local"
)

type CreateLocalUser struct {
	Email        string
	PasswordHash string
	Status       UserStatus
	Role         Role
}

type UpdateUser struct {
	ID        int64
	Email     *string
	Role      *Role
	Status    *UserStatus
	FullName  *string
	Telephone *string
	AvatarURL *string
}

type UserInfo struct {
	ID           int64
	Email        string
	EmailLocked  bool
	Role         Role
	Status       UserStatus
	PasswordHash string
	FullName     *string
	Telephone    *string
	AvatarURL    *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type FindUser struct {
	ID       *int64
	Email    *string
	Role     *Role
	Provider *Provider
	Password *string
	// The maximum number of users to return.
	Limit *int
}

type DeleteUser struct {
	ID int64
}

func (s *Store) CreateLocalUser(ctx context.Context, create *CreateLocalUser) (*UserInfo, error) {
	user, err := s.driver.CreateLocalUser(ctx, create)
	if err != nil {
		if isDuplicateKey(err) {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	s.userCache.Store(user.ID, user)
	return user, nil
}

func (s *Store) UpsertGoogleUser(ctx context.Context, email, sub string) (*UserInfo, error) {
	user, err := s.driver.UpsertGoogleUser(ctx, email, sub)
	if err != nil {
		return nil, err
	}

	s.userCache.Store(user.ID, user)
	return user, nil
}

func (s *Store) UpdateUser(ctx context.Context, update *UpdateUser) (*UserInfo, error) {
	user, err := s.driver.UpdateUser(ctx, update)
	if err != nil {
		return nil, err
	}

	s.userCache.Store(user.ID, user)
	return user, nil
}

func (s *Store) GetUser(ctx context.Context, find *FindUser) (*UserInfo, error) {
	if find.ID != nil {
		if cache, ok := s.userCache.Load(*find.ID); ok {
			return cache.(*UserInfo), nil
		}
	}

	user, err := s.driver.GetUser(ctx, find)
	if err != nil {
		return nil, err
	}

	s.userCache.Store(user.ID, user)
	return user, nil
}

func (s *Store) ListUsers(ctx context.Context, find *FindUser) ([]*UserInfo, error) {
	list, err := s.driver.ListUsers(ctx, find)
	if err != nil {
		return nil, err
	}

	for _, user := range list {
		s.userCache.Store(user.ID, user)
	}
	return list, nil
}

func (s *Store) DeleteUser(ctx context.Context, delete *DeleteUser) (bool, error) {
	s.userCache.Delete(delete.ID)
	return s.driver.DeleteUser(ctx, delete)
}

func isDuplicateKey(err error) bool {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number == 1062 // ER_DUP_ENTRY
	}
	return false
}
