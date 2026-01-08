MIGRATE ?= migrate
MIGRATIONS_DIR := store/db/mysql/migrations

MYSQL_USER ?= appuser
MYSQL_PASSWORD ?= password
MYSQL_HOST ?= 127.0.0.1
MYSQL_PORT ?= 3306
MYSQL_DB ?= appdb

DB_URL := mysql://$(MYSQL_USER):$(MYSQL_PASSWORD)@tcp($(MYSQL_HOST):$(MYSQL_PORT))/$(MYSQL_DB)?multiStatements=true

.PHONY: migrate-create migrate-up migrate-down migrate-force migrate-version

migrate-create:
	@ if [ -z "$(name)" ]; then echo "usage: make migrate-create name=add_feature"; exit 1; fi
	$(MIGRATE) create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

migrate-up:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up

migrate-down:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down 1

migrate-force:
	@ if [ -z "$(version)" ]; then echo "usage: make migrate-force version=N"; exit 1; fi
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" force $(version)

migrate-version:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" version
