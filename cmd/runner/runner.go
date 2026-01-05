package runner

import (
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/sagarsuperuser/userprofile/internal/common"
	"github.com/sagarsuperuser/userprofile/server"
	"github.com/sagarsuperuser/userprofile/server/settings"
	"github.com/sagarsuperuser/userprofile/store"
	"github.com/sagarsuperuser/userprofile/store/db"
)

type Runner struct {
	settings *settings.Settings
	srv      *server.Server
	mu       sync.Mutex
}

func NewRunner(s *settings.Settings) Runner {
	return Runner{
		settings: s,
	}
}

func (runner *Runner) Run() {
	// setup logger
	if err := setupLogger(runner.settings); err != nil {
		log.Fatal().Err(err).Msg("Failed to set up logger")
	}
	log.Info().Str("mode", runner.settings.Mode).
		Str("log_level", log.Logger.GetLevel().String()).
		Msg("Logger initialized")

	// setup Database driver
	dbDriver := db.NewDBDriver(runner.settings, common.NowUTC)

	// set up store
	storeInstance := store.New(dbDriver, common.NowUTC)

	// setup server
	srv := server.NewServer(runner.settings, storeInstance)

	runner.mu.Lock()
	runner.srv = srv
	runner.mu.Unlock()

	srv.Start()

}

func (runner *Runner) Stop() {
	runner.mu.Lock()
	srv := runner.srv
	runner.mu.Unlock()
	if srv != nil {
		srv.Stop()
	}
}
