package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sagarsuperuser/userprofile/errdefs"
	"github.com/sagarsuperuser/userprofile/internal/common"
	"github.com/sagarsuperuser/userprofile/internal/httputil"
	oauth2Utils "github.com/sagarsuperuser/userprofile/internal/oauth2"
	"github.com/sagarsuperuser/userprofile/internal/router"
	sessionUtils "github.com/sagarsuperuser/userprofile/internal/session"
	"github.com/sagarsuperuser/userprofile/store"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
)

type SignupReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *SignupReq) Validate() error {
	username := strings.TrimSpace(s.Username)
	password := strings.TrimSpace(s.Password)
	if username == "" || password == "" {
		return errors.New("username or password is required")
	}

	if err := common.ValidateEmail(username); err != nil {
		return err
	}

	if len(password) < 8 || len(password) > 200 {
		return errors.New("invalid password")
	}

	return nil
}

type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *LoginReq) Validate() error {
	username := strings.TrimSpace(s.Username)
	password := strings.TrimSpace(s.Password)
	if username == "" || password == "" {
		return errors.New("username or password is required")
	}
	return nil
}

type LoginResp struct {
	AuthToken string `json:"auth_token"`
}

func (s *APIV1Service) SignUp(ctx context.Context, rw http.ResponseWriter, req *http.Request, vars map[string]string) error {
	signup := SignupReq{}
	if err := json.NewDecoder(req.Body).Decode(&signup); err != nil {
		return errdefs.InvalidParameter(fmt.Errorf("malformed request body: %w", err))
	}

	if err := signup.Validate(); err != nil {
		return errdefs.InvalidParameter(err)
	}

	role := store.RoleUser
	status := store.StatusActive
	userCreate := &store.CreateLocalUser{
		Email:  signup.Username,
		Status: status,
		Role:   role,
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(signup.Password), bcrypt.DefaultCost)
	if err != nil {
		return errdefs.System(fmt.Errorf("failed to generate password hash: %w", err))
	}

	strHash := string(passwordHash)
	userCreate.PasswordHash = strHash

	user, err := s.Store.CreateLocalUser(ctx, userCreate)
	if err != nil {
		if errors.Is(err, store.ErrUserAlreadyExists) {
			return errdefs.Conflict(errors.New("user is already registered. please login"))
		}

		err = fmt.Errorf("failed to create user: %w", err)
		return errdefs.System(err)
	}

	csResult, err := s.Store.CreateSession(ctx, user.ID)
	if err != nil {
		err = fmt.Errorf("failed to create sesssion: %w", err)
		return errdefs.System(err)
	}
	sessionUtils.SetSessionCookie(rw, req, csResult.Session.ExpiresAt, csResult.Token)

	return httputil.WriteRawJSON(rw, http.StatusOK, newUserResp(user))
}

func (s *APIV1Service) LogIn(ctx context.Context, rw http.ResponseWriter, req *http.Request, vars map[string]string) error {
	loginReq := LoginReq{}
	if err := json.NewDecoder(req.Body).Decode(&loginReq); err != nil {
		return errdefs.InvalidParameter(fmt.Errorf("malformed request body: %w", err))
	}

	if err := loginReq.Validate(); err != nil {
		return errdefs.InvalidParameter(err)
	}

	user, err := s.Store.GetUser(ctx, &store.FindUser{Email: &loginReq.Username})
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			return errdefs.Unauthorized(errors.New("invalid credentials, please try again"))
		}
		return errdefs.System(err)
	}

	// Compare the stored hashed password, with the password that is received.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(loginReq.Password)); err != nil {
		// If the two passwords don't match, return a 401 status.
		return errdefs.Unauthorized(errors.New("invalid credentials, please try again"))
	}

	csResult, err := s.Store.CreateSession(ctx, user.ID)
	if err != nil {
		err = fmt.Errorf("failed to create sesssion: %w", err)
		return errdefs.System(err)
	}
	sessionUtils.SetSessionCookie(rw, req, csResult.Session.ExpiresAt, csResult.Token)
	return httputil.WriteRawJSON(rw, http.StatusOK, newUserResp(user))
}

func (s *APIV1Service) Oauth2Login(ctx context.Context, rw http.ResponseWriter, req *http.Request, vars map[string]string) error {
	verifier := oauth2.GenerateVerifier()
	state := "state"

	if err := oauth2Utils.SetOAuthTempCookie(rw, state, verifier, 5*time.Minute); err != nil {
		err = fmt.Errorf("failed to set oauth temp cookie: %w", err)
		return errdefs.System(err)
	}

	url := s.OAuthConfig.AuthCodeURL(
		state,
		oauth2.AccessTypeOnline,
		oauth2.S256ChallengeOption(verifier),
	)

	http.Redirect(rw, req, url, http.StatusFound)
	return nil
}

func (s *APIV1Service) Oauth2Callback(ctx context.Context, rw http.ResponseWriter, req *http.Request, vars map[string]string) error {
	q := req.URL.Query()
	code := q.Get("code")
	state := q.Get("state")
	if code == "" || state == "" {
		return errdefs.InvalidParameter(errors.New("missing code/state"))
	}

	cookie, err := oauth2Utils.ReadOAuthTempCookie(req)
	if err != nil {
		err := fmt.Errorf("temp cookie read failed: %w", err)
		return errdefs.System(err)
	}

	tok, err := s.OAuthConfig.Exchange(ctx, code, oauth2.VerifierOption(cookie.Verifier))
	if err != nil {
		err = fmt.Errorf("exchange call failed: %w", err)
		return errdefs.Unauthorized(err)
	}

	client := s.OAuthConfig.Client(ctx, tok)
	googleUser, err := oauth2Utils.FetchProviderUserInfo(ctx, client)
	if err != nil {
		err = fmt.Errorf("failed to fetch userinfo: %w", err)
		return errdefs.Unauthorized(err)
	}
	// upsert user in DB
	user, err := s.Store.UpsertGoogleUser(ctx, googleUser.Email, googleUser.Sub)
	if err != nil {
		err = fmt.Errorf("failed to create/update user: %w", err)
		return errdefs.System(err)
	}

	oauth2Utils.ClearOAuthTempCookie(rw)

	// create session token
	csResult, err := s.Store.CreateSession(ctx, user.ID)
	if err != nil {
		err = fmt.Errorf("failed to create sesssion: %w", err)
		return errdefs.System(err)
	}
	sessionUtils.SetSessionCookie(rw, req, csResult.Session.ExpiresAt, csResult.Token)
	return httputil.WriteRawJSON(rw, http.StatusOK, newUserResp(user))
}

func (s *APIV1Service) LogOut(ctx context.Context, rw http.ResponseWriter, req *http.Request, vars map[string]string) error {
	// get session info from context
	sInfo := router.SessionInfoFromContext(ctx)
	if sInfo == nil {
		errdefs.System(errors.New("session info not found in context"))
	}
	_, err := s.Store.RevokeSession(ctx, sInfo)
	if err != nil {
		errdefs.System(fmt.Errorf("failed to revoke session: %w", err))
	}

	return httputil.WriteRawJSON(rw, http.StatusOK, nil)
}
