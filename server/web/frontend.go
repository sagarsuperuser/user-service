package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	sessionUtils "github.com/sagarsuperuser/userprofile/internal/session"
	apiv1 "github.com/sagarsuperuser/userprofile/server/routes/api/v1"
	"github.com/sagarsuperuser/userprofile/server/settings"
)

type Frontend struct {
	templates *TemplateStore
	api       *APIClient
}

type apiError struct {
	Message string `json:"message"`
}

type loginPageData struct {
	Title string
	Error string
}

type signupPageData struct {
	Title string
	Error string
}

type profilePageData struct {
	Title string
	User  *apiv1.UserResp
}

type profileFormData struct {
	Title        string
	User         *apiv1.UserResp
	Error        string
	DisableEmail bool
}

type ctxUserKey struct{}

const flashCookieName = "ui_flash"
const flashMaxAgeSeconds = 60

func NewFrontend(s *settings.Settings) (*Frontend, error) {
	templates, err := NewTemplateStore()
	if err != nil {
		return nil, err
	}

	return &Frontend{
		templates: templates,
		api:       NewAPIClient(s),
	}, nil
}

func (f *Frontend) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/", f.loginPage).Methods(http.MethodGet)
	r.HandleFunc("/login", f.loginPage).Methods(http.MethodGet)
	r.HandleFunc("/login", f.handleLogin).Methods(http.MethodPost)

	r.HandleFunc("/signup", f.signupPage).Methods(http.MethodGet)
	r.HandleFunc("/signup", f.handleSignup).Methods(http.MethodPost)

	auth := f.authenticated
	r.HandleFunc("/profile", auth(f.profilePage)).Methods(http.MethodGet)
	r.HandleFunc("/profile/edit", auth(f.editProfilePage)).Methods(http.MethodGet)
	r.HandleFunc("/profile/edit", auth(f.handleProfileUpdate)).Methods(http.MethodPost)

	r.HandleFunc("/logout", f.handleLogout).Methods(http.MethodPost)

	// OAuth callback endpoint - forwards to API then redirects to profile
	r.HandleFunc("/ui/oauth2/callback", f.handleOAuthCallback).Methods(http.MethodGet)
}

func (f *Frontend) loginPage(w http.ResponseWriter, r *http.Request) {
	user, status, err := f.api.FetchCurrentUser(r.Context(), r)
	if err != nil {
		f.serverError(w, err)
		return
	}
	if status == http.StatusOK && user != nil {
		http.Redirect(w, r, "/profile", http.StatusFound)
		return
	}

	f.templates.Render(w, "login.html", loginPageData{
		Title: "Login",
		Error: f.popFlash(w, r),
	})
}

func (f *Frontend) signupPage(w http.ResponseWriter, r *http.Request) {
	user, status, err := f.api.FetchCurrentUser(r.Context(), r)
	if err != nil {
		f.serverError(w, err)
		return
	}
	if status == http.StatusOK && user != nil {
		http.Redirect(w, r, "/profile", http.StatusFound)
		return
	}

	f.templates.Render(w, "signup.html", signupPageData{
		Title: "Sign Up",
		Error: f.popFlash(w, r),
	})
}

func (f *Frontend) profilePage(w http.ResponseWriter, r *http.Request) {
	user := currentUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	setNoCacheHeaders(w)
	f.templates.Render(w, "profile.html", profilePageData{
		Title: "Profile",
		User:  user,
	})
}

func (f *Frontend) editProfilePage(w http.ResponseWriter, r *http.Request) {
	user := currentUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	setNoCacheHeaders(w)
	f.templates.Render(w, "profile_edit.html", profileFormData{
		Title:        "Edit Profile",
		User:         user,
		DisableEmail: user.EmailLocked,
		Error:        f.popFlash(w, r),
	})
}

func (f *Frontend) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		f.setFlash(w, "Malformed form submission")
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	payload := map[string]string{
		"username": strings.TrimSpace(r.FormValue("username")),
		"password": r.FormValue("password"),
	}
	resp, err := f.api.Request(r.Context(), r, http.MethodPost, "/auth/login", payload)
	if err != nil {
		f.serverError(w, err)
		return
	}
	defer resp.Body.Close()

	f.copySetCookies(w, resp)
	if resp.StatusCode != http.StatusOK {
		msg := readAPIMessage(resp, "Invalid credentials, please try again")
		f.setFlash(w, msg)
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusFound)
}

func (f *Frontend) handleSignup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		f.setFlash(w, "Malformed form submission")
		http.Redirect(w, r, "/signup", http.StatusFound)
		return
	}
	payload := map[string]string{
		"username": strings.TrimSpace(r.FormValue("username")),
		"password": r.FormValue("password"),
	}
	resp, err := f.api.Request(r.Context(), r, http.MethodPost, "/auth/signup", payload)
	if err != nil {
		f.serverError(w, err)
		return
	}
	defer resp.Body.Close()

	f.copySetCookies(w, resp)
	if resp.StatusCode != http.StatusOK {
		msg := readAPIMessage(resp, "Unable to sign you up")
		f.setFlash(w, msg)
		http.Redirect(w, r, "/signup", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/profile/edit", http.StatusFound)
}

func (f *Frontend) handleProfileUpdate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		f.serverError(w, err)
		return
	}
	user := currentUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	payload := struct {
		Email     *string `json:"email,omitempty"`
		FullName  *string `json:"full_name,omitempty"`
		Telephone *string `json:"telephone,omitempty"`
	}{
		Email:     stringPtr(strings.TrimSpace(r.FormValue("email"))),
		FullName:  stringPtr(strings.TrimSpace(r.FormValue("full_name"))),
		Telephone: stringPtr(strings.TrimSpace(r.FormValue("telephone"))),
	}
	// keep locked email untouched
	if user.EmailLocked {
		payload.Email = nil
	}

	resp, err := f.api.Request(r.Context(), r, http.MethodPatch, "/user", payload)
	if err != nil {
		f.serverError(w, err)
		return
	}
	defer resp.Body.Close()

	f.copySetCookies(w, resp)

	if resp.StatusCode != http.StatusOK {
		f.setFlash(w, readAPIMessage(resp, "Unable to update profile"))
		http.Redirect(w, r, "/profile/edit", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusFound)
}

func (f *Frontend) handleLogout(w http.ResponseWriter, r *http.Request) {
	resp, err := f.api.Request(r.Context(), r, http.MethodGet, "/auth/logout", nil)
	if err == nil && resp != nil {
		f.copySetCookies(w, resp)
		resp.Body.Close()
	}

	// Clear session cookie locally to be safe
	http.SetCookie(w, &http.Cookie{
		Name:     sessionUtils.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

func (f *Frontend) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	endpoint := "/oauth2/callback"
	if raw := r.URL.RawQuery; raw != "" {
		endpoint = endpoint + "?" + raw
	}
	resp, err := f.api.Request(r.Context(), r, http.MethodGet, endpoint, nil)
	if err != nil {
		f.serverError(w, err)
		return
	}
	defer resp.Body.Close()

	f.copySetCookies(w, resp)
	if resp.StatusCode != http.StatusOK {
		f.setFlash(w, readAPIMessage(resp, "Google sign-in failed"))
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusFound)
}

// middleware
func (f *Frontend) authenticated(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, status, err := f.api.FetchCurrentUser(r.Context(), r)
		if err != nil {
			f.serverError(w, err)
			return
		}
		if status == http.StatusUnauthorized || user == nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserKey{}, user)
		next(w, r.WithContext(ctx))
	}
}

func (f *Frontend) popFlash(w http.ResponseWriter, r *http.Request) string {
	c, err := r.Cookie(flashCookieName)
	if err != nil {
		return ""
	}
	// clear
	http.SetCookie(w, &http.Cookie{
		Name:     flashCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	val, _ := url.QueryUnescape(c.Value)
	return val
}

func (f *Frontend) setFlash(w http.ResponseWriter, msg string) {
	if strings.TrimSpace(msg) == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     flashCookieName,
		Value:    url.QueryEscape(msg),
		Path:     "/",
		MaxAge:   flashMaxAgeSeconds,
		HttpOnly: true,
	})
}

func (f *Frontend) serverError(w http.ResponseWriter, err error) {
	log.Error().Err(err).Msg("frontend error")
	http.Error(w, "unexpected server error", http.StatusInternalServerError)
}

func (f *Frontend) copySetCookies(w http.ResponseWriter, resp *http.Response) {
	for _, c := range resp.Cookies() {
		http.SetCookie(w, c)
	}
}

func readAPIMessage(resp *http.Response, fallback string) string {
	body, err := io.ReadAll(resp.Body)
	if err != nil || len(body) == 0 {
		return fallback
	}

	var apiErr apiError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		return fallback
	}
	if strings.TrimSpace(apiErr.Message) == "" {
		return fallback
	}
	return apiErr.Message
}

func stringPtr(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	s := v
	return &s
}

func currentUserFromContext(ctx context.Context) *apiv1.UserResp {
	if ctx == nil {
		return nil
	}
	if val := ctx.Value(ctxUserKey{}); val != nil {
		return val.(*apiv1.UserResp)
	}
	return nil
}

func setNoCacheHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Cache-Control", "no-store")
}
