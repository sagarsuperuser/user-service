package settings

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Settings is the configuration to start main server.
type Settings struct {
	// Mode can be "prod" or "dev"
	Mode string `envconfig:"MODE" default:"dev"`

	// Server listen address config
	Host string `envconfig:"HOST" default:"0.0.0.0"`
	Port int    `envconfig:"PORT" default:"8080"`

	// TODO: enable debug server
	// DebugHost string `envconfig:"DEBUG_HOST" default:"0.0.0.0"`
	// DebugPort int    `envconfig:"DEBUG_PORT" default:"6070"`

	// Driver is the database driver
	// mysql only supported for now
	Driver string `envconfig:"DRIVER" default:"mysql"`

	// MySQL settings
	MySQLHost     string `envconfig:"MYSQL_HOST" default:"127.0.0.1"`
	MySQLPort     int    `envconfig:"MYSQL_PORT" default:"3306"`
	MySQLDatabase string `envconfig:"MYSQL_DB" default:"appdb"`
	MySQLUser     string `envconfig:"MYSQL_USER" default:"appuser"`
	MySQLPassword string `envconfig:"MYSQL_PASSWORD" default:"password"`
	// Timeouts
	MySQLConnectTimeout time.Duration `envconfig:"MYSQL_CONNECT_TIMEOUT" default:"5s"`
	MySQLQueryTimeout   time.Duration `envconfig:"MYSQL_QUERY_TIMEOUT" default:"5s"`
	// Pool
	MySQLMaxOpenConns    int           `envconfig:"MYSQL_MAX_OPEN_CONNS" default:"25"`
	MySQLMaxIdleConns    int           `envconfig:"MYSQL_MAX_IDLE_CONNS" default:"10"`
	MySQLConnMaxLifetime time.Duration `envconfig:"MYSQL_CONN_MAX_LIFETIME" default:"30m"`
	MySQLConnMaxIdleTime time.Duration `envconfig:"MYSQL_CONN_MAX_IDLE_TIME" default:"5m"`

	// Logging settings
	LogLevel  string `envconfig:"LOG_LEVEL" default:"debug"`
	LogFormat string `envconfig:"LOG_FORMAT" default:"text"`

	// Secret key used for signing JWT tokens
	SecretKey string `envconfig:"SECRET_KEY" default:"secretkey"`

	// Origins is the list of allowed origins
	Origins []string `envconfig:"ORIGINS" default:""`
	// OAuth2 settings
	OAuth2ClientID     string `envconfig:"OAUTH2_CLIENT_ID" default:""`
	OAuth2ClientSecret string `envconfig:"OAUTH2_CLIENT_SECRET" default:""`
	OAuth2RedirectURL  string `envconfig:"OAUTH2_REDIRECT_URL" default:""`
}

// NewSettings loads settings  by reading environment variables.
func NewSettings() *Settings {
	s := new(Settings)
	if err := envconfig.Process("", s); err != nil {
		panic(err)
	}

	return s
}
