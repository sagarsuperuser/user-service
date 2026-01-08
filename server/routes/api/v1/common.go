package v1

import (
	"time"

	"github.com/sagarsuperuser/userprofile/store"
)

type UserResp struct {
	ID          int64            `json:"id"`
	Email       string           `json:"email"`
	EmailLocked bool             `json:"email_locked"`
	Status      store.UserStatus `json:"status"`
	Role        store.Role       `json:"role"`
	FullName    string           `json:"full_name"`
	Telephone   string           `json:"telephone"`
	AvatarURL   string           `json:"avatar_url"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

func newUserResp(u *store.UserInfo) *UserResp {
	if u == nil {
		return nil
	}
	resp := &UserResp{
		ID:          u.ID,
		Email:       u.Email,
		EmailLocked: u.EmailLocked,
		Role:        u.Role,
		Status:      u.Status,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
	if u.FullName != nil {
		resp.FullName = *u.FullName
	}
	if u.Telephone != nil {
		resp.Telephone = *u.Telephone
	}
	if u.AvatarURL != nil {
		resp.AvatarURL = *u.AvatarURL
	}

	return resp
}
