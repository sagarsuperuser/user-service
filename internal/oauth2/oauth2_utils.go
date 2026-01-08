package oauth2Utils

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/sagarsuperuser/userprofile/server/settings"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	// __Host- cookies must be Secure, Path=/, and must not include Domain.
	// https://developer.mozilla.org/en-US/docs/Web/API/Document/cookie#cookie_prefixes
	OauthTempCookieName    = "__Host-oauth_tmp"
	GoogleUserInfoEndpoint = "https://www.googleapis.com/oauth2/v3/userinfo"
)

type oauthTemp struct {
	State    string `json:"state"`
	Verifier string `json:"verifier"`
}

type ProviderUserInfoResp struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// TODO: support multiple providers
// NewOAuth2Config initializes a new OAuth2 Config for Google provider.
func NewOAuth2Config(s *settings.Settings) *oauth2.Config {
	u, err := url.Parse(s.OAuth2RedirectURL)
	if err != nil {
		log.Fatal().Err(err).Msg("OAuth2Config failed to parse redirect url")
	}
	return &oauth2.Config{
		ClientID:     s.OAuth2ClientID,
		ClientSecret: s.OAuth2ClientSecret,
		RedirectURL:  u.String(),
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

func SetOAuthTempCookie(w http.ResponseWriter, state, verifier string, ttl time.Duration) error {
	v := oauthTemp{
		State:    state,
		Verifier: verifier,
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     OauthTempCookieName,
		Value:    base64.RawURLEncoding.EncodeToString(raw),
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // end to end oauth2 flow only works for https
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	})
	return nil
}

func ReadOAuthTempCookie(r *http.Request) (*oauthTemp, error) {
	c, err := r.Cookie(OauthTempCookieName)
	if err != nil {
		return nil, err
	}
	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		return nil, err
	}
	ret := new(oauthTemp)
	if err := json.Unmarshal(raw, ret); err != nil {
		return nil, err
	}
	if ret.State == "" || ret.Verifier == "" {
		return nil, fmt.Errorf("invalid oauth temp cookie")
	}
	return ret, nil
}

func ClearOAuthTempCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     OauthTempCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // end to end oauth2 flow only works for https
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// client underlying HTTP transport is responsible for adding access token to request.
func FetchProviderUserInfo(ctx context.Context, client *http.Client) (*ProviderUserInfoResp, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, GoogleUserInfoEndpoint, nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Userinfo status %d", resp.StatusCode)
	}
	var user ProviderUserInfoResp
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	if user.Email == "" {
		return nil, fmt.Errorf("Missing email")
	}
	return &user, nil
}
