package db

import (
	"github.com/rs/zerolog/log"

	"github.com/sagarsuperuser/userprofile/internal/common"
	"github.com/sagarsuperuser/userprofile/server/settings"
	"github.com/sagarsuperuser/userprofile/store"
	"github.com/sagarsuperuser/userprofile/store/db/mysql"
)

// NewDBDriver creates new db driver based on settings.
func NewDBDriver(settings *settings.Settings, now common.NowFunc) store.Driver {
	var driver store.Driver

	switch settings.Driver {
	case "mysql":
		driver = mysql.NewDB(settings, now)

	default:
		log.Fatal().Str("driver", settings.Driver).Msg("Unsupported DB driver")
	}
	return driver
}
