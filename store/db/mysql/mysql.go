package mysql

import (
	"database/sql"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog/log"
	"github.com/sagarsuperuser/userprofile/internal/common"
	"github.com/sagarsuperuser/userprofile/server/settings"
	"github.com/sagarsuperuser/userprofile/store"
)

type DB struct {
	db       *sql.DB
	settings *settings.Settings
	config   *mysql.Config
	now      common.NowFunc
}

func NewDB(settings *settings.Settings, now common.NowFunc) store.Driver {
	driver := DB{settings: settings}
	driver.config = createConfig(settings)
	dsn := driver.config.FormatDSN()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open MySQL connection")
	}
	driver.db = db

	// set pool options
	driver.db.SetMaxOpenConns(settings.MySQLMaxOpenConns)
	driver.db.SetMaxIdleConns(settings.MySQLMaxIdleConns)
	driver.db.SetConnMaxLifetime(settings.MySQLConnMaxLifetime)
	driver.db.SetConnMaxIdleTime(settings.MySQLConnMaxIdleTime)

	// Test the connection
	if err := driver.db.Ping(); err != nil {
		log.Debug().Str("dsn", dsn).Msg("Configured DSN")
		log.Fatal().Err(err).Msg("Failed to ping MySQL")
	}

	log.Info().
		Str("host", settings.MySQLHost).
		Int("port", settings.MySQLPort).
		Str("database", settings.MySQLDatabase).
		Msg("Connected to MySQL")

	driver.now = now
	return &driver
}

func (d *DB) GetDB() *sql.DB {
	return d.db
}

func (d *DB) Close() error {
	return d.db.Close()
}

func createConfig(settings *settings.Settings) *mysql.Config {
	config := mysql.NewConfig()
	config.User = settings.MySQLUser
	config.Passwd = settings.MySQLPassword
	config.Net = "tcp"
	config.Addr = fmt.Sprintf("%s:%d", settings.MySQLHost, settings.MySQLPort)
	config.DBName = settings.MySQLDatabase
	// Open MySQL connection with parameter.
	// multiStatements=true is required for migration.
	// See more in: https://github.com/go-sql-driver/mysql#multistatements
	config.MultiStatements = true
	config.ParseTime = true
	// Timeouts
	config.Timeout = settings.MySQLConnectTimeout
	config.ReadTimeout = settings.MySQLQueryTimeout
	config.WriteTimeout = settings.MySQLQueryTimeout
	return config

}
