package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	migrate "github.com/golang-migrate/migrate/v4"
	migrateMySQL "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/sagarsuperuser/userprofile/cmd/runner"
	"github.com/sagarsuperuser/userprofile/internal/common"
	"github.com/sagarsuperuser/userprofile/server/settings"
	mysqlDriver "github.com/sagarsuperuser/userprofile/store/db/mysql"
)

func main() {
	settings := settings.NewSettings()
	args := os.Args
	cmd := "serve"
	if len(args) > 1 {
		cmd = strings.ToLower(args[1])
	}

	switch cmd {
	case "serve":
		runServer(settings)
	case "migrate":
		if err := runMigrations(settings); err != nil {
			log.Fatalf("migration failed: %v", err)
		}
	default:
		log.Fatalf("unknown command %q (use serve|migrate)", cmd)
	}
}

func runServer(settings *settings.Settings) {
	r := runner.NewRunner(settings)
	r.Run()
}

func runMigrations(settings *settings.Settings) error {
	drv := mysqlDriver.NewDB(settings, common.NowUTC).(*mysqlDriver.DB)
	defer drv.Close()

	inst, err := migrateMySQL.WithInstance(drv.GetDB(), &migrateMySQL.Config{})
	if err != nil {
		return fmt.Errorf("init migrate instance: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://store/db/mysql/migrations", "mysql", inst)
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("apply migrations: %w", err)
	}
	log.Printf("migrations applied (source: file://store/db/mysql/migrations)")
	return nil
}
