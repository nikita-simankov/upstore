.PHONY: proto sqlc-generate migrate-accounts-up migrate-accounts-down migrate-profiles-up migrate-profiles-down

DOCKER_NETWORK_NAME := upstore
DOCKER_COMPOSE_INFRA := docker compose -f infra/docker-compose.infra.yaml
DOCKER_COMPOSE_SERVICES := docker compose -f infra/docker-compose.services.yaml

proto:
	@echo "[make:proto] Generating code..."
	@find proto -name '*.proto' -not -path 'proto/generated/*' | \
		xargs protoc \
			-I proto \
			--go_out=paths=source_relative:proto/generated \
			--go-grpc_out=paths=source_relative:proto/generated
	@echo "[make:proto] Code generation complete"

docker-network-create:
	@docker network inspect $(DOCKER_NETWORK_NAME) >/dev/null 2>&1 || \
		(echo "[make:docker-network-create] Creating network $(DOCKER_NETWORK_NAME)..." && \
		docker network create $(DOCKER_NETWORK_NAME))

compose-infra-up: docker-network-create
	$(DOCKER_COMPOSE_INFRA) up -d
	@echo "[make:compose-infra-up] Infrastructure is up. Services will be healthy in ~10s."

compose-infra-down:
	$(DOCKER_COMPOSE_INFRA) down

compose-infra-logs:
	$(DOCKER_COMPOSE_INFRA) logs -f

compose-infra-restart:
	$(DOCKER_COMPOSE_INFRA) down -v
	$(DOCKER_COMPOSE_INFRA) up -d
	@echo "[make:compose-infra-restart] Infrastructure restart complete"

compose-services-up: docker-network-create
	$(DOCKER_COMPOSE_SERVICES) up -d --build

compose-services-down:
	$(DOCKER_COMPOSE_SERVICES) down

compose-services-logs:
	$(DOCKER_COMPOSE_SERVICES) logs -f

compose-services-build:
	$(DOCKER_COMPOSE_SERVICES) build

sqlc-generate:
	@echo "[make:sqlc-generate] Generating database code..."
	sqlc generate -f services/accounts/sqlc.yaml
	sqlc generate -f services/profiles/sqlc.yaml
	@echo "[make:sqlc-generate] Database code generation complete"


DATABASE_OWNER_URL ?= postgres://upstore_owner:upstore_owner_pass@127.0.0.1:5432/upstore?sslmode=disable

# Each service keeps its own goose version table, so both migration sets can share one database.
migrate-accounts-up:
	goose -table accounts_schema_version -dir services/accounts/migrations postgres "$(DATABASE_OWNER_URL)" up

migrate-accounts-down:
	goose -table accounts_schema_version -dir services/accounts/migrations postgres "$(DATABASE_OWNER_URL)" down

migrate-profiles-up:
	goose -table profiles_schema_version -dir services/profiles/migrations postgres "$(DATABASE_OWNER_URL)" up

migrate-profiles-down:
	goose -table profiles_schema_version -dir services/profiles/migrations postgres "$(DATABASE_OWNER_URL)" down
