.PHONY: proto

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

compose-services-build:
	$(DOCKER_COMPOSE_SERVICES) build