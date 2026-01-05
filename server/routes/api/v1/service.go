package v1

import (
	oauth2Utils "github.com/sagarsuperuser/userprofile/internal/oauth2"
	"github.com/sagarsuperuser/userprofile/server/settings"
	"github.com/sagarsuperuser/userprofile/store"
	"golang.org/x/oauth2"
)

// APIV1Service holds shared dependencies for v1 routes.
type APIV1Service struct {
	Settings    *settings.Settings
	Store       *store.Store
	OAuthConfig *oauth2.Config
}

func NewAPIV1Service(s *settings.Settings, store *store.Store) *APIV1Service {
	return &APIV1Service{
		Settings:    s,
		Store:       store,
		OAuthConfig: oauth2Utils.NewOAuth2Config(s),
	}
}
