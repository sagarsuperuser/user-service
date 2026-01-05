MIGRATE ?= migrate
MIGRATIONS_DIR := migrations

DB_USER ?= root
DB_PASS ?= password
DB_HOST ?= 127.0.0.1
DB_PORT ?= 3306
DB_NAME ?= appdb

DB_URL := mysql://$(DB_USER):$(DB_PASS)@tcp($(DB_HOST):$(DB_PORT))/$(DB_NAME)?multiStatements=true

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
