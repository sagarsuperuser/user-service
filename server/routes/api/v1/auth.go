package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/rs/zerolog/hlog"
	"github.com/sagarsuperuser/userprofile/errdefs"
	"github.com/sagarsuperuser/userprofile/internal/httputil"
	jwtUtils "github.com/sagarsuperuser/userprofile/internal/jwt"
	oauth2Utils "github.com/sagarsuperuser/userprofile/internal/oauth2"
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
	if username == "" {
		return errors.New("invalid username")
	}

	if len(username) > 254 {
		return errors.New("invalid username")
	}

	email := strings.ToLower(username)

	// only email is alllowed as username
	_, err := mail.ParseAddress(email)
	if err != nil {
		return errors.New("invalid username")
	}

	if len(s.Password) < 8 || len(s.Password) > 200 {
		return errors.New("invalid password")
	}

	return nil
}

type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
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
	userCreate := &store.CreateUser{
		Email:       signup.Username,
		EmailLocked: false,
		Status:      &status,
		Role:        &role,
		Provider:    store.ProviderLocal,
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(signup.Password), bcrypt.DefaultCost)
	if err != nil {
		return errdefs.System(fmt.Errorf("failed to generate password hash: %w", err))
	}

	strHash := string(passwordHash)
	userCreate.PasswordHash = &strHash

	user, err := s.Store.CreateUser(ctx, userCreate)
	if err != nil {
		err = fmt.Errorf("failed to create user: %w", err)
		return errdefs.System(err)
	}

	csResult, err := s.Store.CreateSession(ctx, user.ID)
	if err != nil {
		err = fmt.Errorf("failed to create sesssion: %w", err)
		return errdefs.System(err)
	}
	sessionUtils.SetSessionCookie(rw, req, csResult.Session.ExpiresAt, csResult.Token)
	return nil
}

func (s *APIV1Service) LogIn(ctx context.Context, rw http.ResponseWriter, req *http.Request, vars map[string]string) error {
	return httputil.WriteRawJSON(rw, http.StatusOK, LoginResp{
		AuthToken: "",
	})
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
	user, err := oauth2Utils.FetchProviderUserInfo(ctx, client)
	if err != nil {
		err = fmt.Errorf("failed to fetch userinfo: %w", err)
		return errdefs.Unauthorized(err)
	}
	hlog.FromRequest(req).Info().Any("user", user).Send()

	token, err := jwtUtils.GenerateAccessToken(user.Email, 123456789, []byte(s.Settings.SecretKey))
	if err != nil {
		err := fmt.Errorf("failed to generate access token: %w", err)
		return errdefs.System(err)
	}

	oauth2Utils.ClearOAuthTempCookie(rw)

	return httputil.WriteRawJSON(rw, http.StatusOK, LoginResp{
		AuthToken: token,
	})
}

func (s *APIV1Service) LogOut(ctx context.Context, rw http.ResponseWriter, req *http.Request, vars map[string]string) error {
	return httputil.WriteRawJSON(rw, http.StatusOK, nil)
}
