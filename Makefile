.DEFAULT_GOAL := help

COMPOSE := sudo docker compose --project-directory infra --env-file infra/.env -f infra/docker-compose.yaml
COMPOSE_DEV := $(COMPOSE) -f infra/docker-compose.dev.yaml
COMPOSE_TEST := $(COMPOSE) -f infra/docker-compose.test.yaml
COMPOSE_PROD := $(COMPOSE) -f infra/docker-compose.prod.yaml
DOCKERFILE := infra/Dockerfile
IMAGE := twivo-core

.PHONY: help test dev prod down dev-down test-down dev-volumes-remove logs shell shell-test build build-dev build-test build-prod

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "%-12s %s\n", $$1, $$2}'

test: ## Run Go tests in Docker with ephemeral MongoDB and Redis
	$(COMPOSE_TEST) run --build --rm twivo-core go test ./...

dev: ## Start the development app, MongoDB, and Redis
	$(COMPOSE_DEV) up -d --build mongodb redis twivo-core

prod: ## Start the production app, MongoDB, and Redis
	$(COMPOSE_PROD) up -d --build mongodb redis twivo-core

down: dev-down ## Stop development app and database containers

dev-down: ## Stop development app and database containers
	$(COMPOSE_DEV) down

test-down: ## Stop test containers
	$(COMPOSE_TEST) down

dev-volumes-remove: dev-down ## Stop dev containers and permanently remove dev data volumes
	@for volume in twivo-core-dev-mongo-data twivo-core-dev-redis-data twivo-core-dev-nginx-conf; do \
		if sudo docker volume inspect "$$volume" >/dev/null 2>&1; then \
			sudo docker volume rm "$$volume"; \
		else \
			echo "Volume $$volume not found; skipping"; \
		fi; \
	done

logs: ## Follow development app and database logs
	$(COMPOSE_DEV) logs -f --tail=100 twivo-core mongodb redis

shell: ## Open a Bash shell in the running development app container
	$(COMPOSE_DEV) exec twivo-core bash

shell-test: ## Open a Bash shell in a one-off test app container
	$(COMPOSE_TEST) run --rm twivo-core bash

build: build-prod ## Build the production image

build-dev: ## Build the development image
	docker build --file $(DOCKERFILE) --target dev --tag $(IMAGE):dev .

build-test: ## Build the test (development) image
	docker build --file $(DOCKERFILE) --target dev --tag $(IMAGE):test .

build-prod: ## Build the production image
	docker build --file $(DOCKERFILE) --target prod --tag $(IMAGE):prod .