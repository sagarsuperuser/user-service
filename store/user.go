package store

import (
	"context"
	"time"
)

// Role is the type of a role.
type Role string

const (
	// RoleAdmin is the ADMIN role.
	RoleAdmin Role = "ADMIN"
	// RoleUser is the USER role.
	RoleUser Role = "USER"
)

func (e Role) String() string {
	switch e {
	case RoleAdmin:
		return "ADMIN"
	case RoleUser:
		return "USER"
	default:
		return "UNKNOWN"
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

type CreateUser struct {
	ID              int64
	Email           string
	EmailLocked     bool
	Status          *UserStatus
	Role            *Role
	Provider        Provider
	ProviderSubject *string
	PasswordHash    *string
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
	UpdateUser
	EmailLocked bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type FindUser struct {
	ID    *int64
	Email *string
	Role  *Role

	// The maximum number of users to return.
	Limit *int
}

type DeleteUser struct {
	ID int64
}

func (s *Store) CreateUser(ctx context.Context, create *CreateUser) (*UserInfo, error) {
	user, err := s.driver.CreateUser(ctx, create)
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

func (s *Store) GetUser(ctx context.Context, find *FindUser) (*UserInfo, error) {
	if find.ID != nil {
		if cache, ok := s.userCache.Load(*find.ID); ok {
			return cache.(*UserInfo), nil
		}
	}

	list, err := s.ListUsers(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}

	user := list[0]
	s.userCache.Store(user.ID, user)
	return user, nil
}

func (s *Store) DeleteUser(ctx context.Context, delete *DeleteUser) (bool, error) {
	s.userCache.Delete(delete.ID)
	return s.driver.DeleteUser(ctx, delete)
}
