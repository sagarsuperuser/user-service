package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/sagarsuperuser/userprofile/errdefs"
	"github.com/sagarsuperuser/userprofile/internal/common"
	"github.com/sagarsuperuser/userprofile/internal/httputil"
	"github.com/sagarsuperuser/userprofile/internal/router"
	"github.com/sagarsuperuser/userprofile/store"
)

type UpdateUserRequest struct {
	Email     *string `json:"email"`
	FullName  *string `json:"full_name"`
	Telephone *string `json:"telephone"`
	AvatarURL *string `json:"avatar_url"`
}

func (update UpdateUserRequest) Validate() error {
	if update.Email != nil {
		if err := common.ValidateEmail(*update.Email); err != nil {
			return err
		}
	}

	if update.FullName != nil && *update.FullName != "" {
		if len(*update.FullName) > 100 {
			return errors.New("full name is too long, maximum length is 100")
		}
	}

	if update.Telephone != nil {
		if err := common.ValidateTelephone(*update.Telephone); err != nil {
			return err
		}
	}

	if update.AvatarURL != nil {
		if len(*update.AvatarURL) > 200 {
			return errors.New("avatar is too large, maximum length is 200")
		}
	}

	return nil
}

// GetUserList godoc
//
//	@Summary	Get logged in user
//	@Tags		user
//	@Produce	json
//	@Success	200	{object}	store.User	    "User"
//	@Failure	500	{object}	nil				"Failed to fetch user"
//	@Router		/v1/user/me [GET]
func (s *APIV1Service) GetCurrentUser(ctx context.Context, rw http.ResponseWriter, req *http.Request, vars map[string]string) error {
	sInfo := router.SessionInfoFromContext(ctx)
	if sInfo == nil {
		return errdefs.System(errors.New("session info not found in context"))
	}
	// get user by user id.
	user, err := s.Store.GetUser(ctx, &store.FindUser{ID: &sInfo.UserID})
	if err != nil {
		return errdefs.System(err)
	}
	if user == nil {
		return errdefs.System(errors.New("invalid user id in session info. FIX ME!!!"))
	}

	return httputil.WriteRawJSON(rw, http.StatusOK, newUserResp(user))
}

func (s *APIV1Service) UpdateUser(ctx context.Context, rw http.ResponseWriter, req *http.Request, vars map[string]string) error {
	uReq := &UpdateUserRequest{}
	if err := json.NewDecoder(req.Body).Decode(uReq); err != nil {
		return errdefs.InvalidParameter(fmt.Errorf("malformed request body: %w", err))
	}
	if err := uReq.Validate(); err != nil {
		return errdefs.InvalidParameter(err)
	}

	sInfo := router.SessionInfoFromContext(ctx)
	if sInfo == nil {
		return errdefs.Unauthorized(errors.New("session info not found in context"))
	}

	userUpdate := &store.UpdateUser{
		ID:        sInfo.UserID,
		Email:     uReq.Email,
		FullName:  uReq.FullName,
		Telephone: uReq.Telephone,
		AvatarURL: uReq.AvatarURL,
	}

	user, err := s.Store.UpdateUser(ctx, userUpdate)
	if err != nil {
		if errors.Is(err, store.ErrEmailUpdateNotAllowed) {
			return errdefs.Conflict(err)
		}
		return errdefs.System(fmt.Errorf("failed to update user: %w", err))
	}
	return httputil.WriteRawJSON(rw, http.StatusOK, newUserResp(user))

}
