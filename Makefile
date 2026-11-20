# HUBX local platform. Run from Git Bash (Windows) or any POSIX shell; requires Docker, Node and curl.
# Owner: devops. Secrets live in .env (git-ignored), generated on first use.

.DEFAULT_GOAL := help
COMPOSE := docker compose

.PHONY: help env up down reset logs ps migrate psql test test-integration security

help: ## List the targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-18s %s\n", $$1, $$2}'

env: ## Create .env with random local secrets (only if missing)
	@bash scripts/env.sh

up: env ## Start PostgreSQL, run migrations, start API and gateway; returns when /health answers
	$(COMPOSE) up -d --build
	@bash scripts/wait-health.sh http://localhost:8080 90

down: ## Stop the local stack (keeps the database)
	$(COMPOSE) down --remove-orphans

reset: ## Stop the local stack and delete the database volume
	$(COMPOSE) down -v --remove-orphans

logs: ## Follow API, gateway and migration logs
	$(COMPOSE) logs -f migrate api gw

ps: ## Show the local stack
	$(COMPOSE) ps -a

migrate: env ## Apply pending migrations to the running database
	$(COMPOSE) run --rm --build migrate

psql: ## Open psql on the local database (used by the ledger tamper test)
	$(COMPOSE) exec postgres psql -U hubx -d hubx

test: ## Unit tests of every service
	go -C hubx-api test ./...
	go -C hubx-gw test ./...
	npm --prefix hubx-mfe run test --if-present

test-integration: ## Clean isolated stack + hubx-bender through the gateway, then tear down
	@bash scripts/test-integration.sh

security: ## gosec and govulncheck on both Go services
	cd hubx-api && gosec ./...
	cd hubx-gw && gosec ./...
	govulncheck -C hubx-api ./...
	govulncheck -C hubx-gw ./...
