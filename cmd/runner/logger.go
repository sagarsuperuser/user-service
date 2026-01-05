package runner

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"

	"github.com/sagarsuperuser/userprofile/server/settings"
)

func setupLogger(settings *settings.Settings) error {
	// configure log format
	w := getLogWriter(settings)

	// configure log level
	logLevel := getLogLevel(settings)
	zerolog.SetGlobalLevel(logLevel)

	// create logger
	logger := zerolog.New(w).With().Timestamp()
	if logLevel <= zerolog.DebugLevel {
		logger = logger.Caller()
	}

	log.Logger = logger.Logger().Level(logLevel)

	zerolog.DefaultContextLogger = &log.Logger
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	return nil
}

func getLogLevel(settings *settings.Settings) zerolog.Level {
	var levelStr string
	if settings.LogLevel != "" {
		levelStr = strings.ToLower(settings.LogLevel)
	}

	logLevel, err := zerolog.ParseLevel(strings.ToLower(levelStr))
	if err != nil {
		log.Error().Err(err).
			Str("logLevel", levelStr).
			Msg("Unspecified or invalid log level, setting the level to default (ERROR)...")

		logLevel = zerolog.ErrorLevel
	}

	return logLevel
}

func getLogWriter(settings *settings.Settings) io.Writer {
	var w io.Writer = os.Stdout
	useConsole := strings.ToLower(settings.LogFormat) == "text" || settings.Mode != "prod"
	if useConsole {
		w = zerolog.ConsoleWriter{
			Out:        w,
			TimeFormat: time.RFC3339,
		}
	}

	return w

}
