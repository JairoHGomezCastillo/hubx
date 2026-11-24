# HUBX local platform. Run from Git Bash (Windows) or any POSIX shell; requires Docker, Node and curl.
# Owner: devops. Secrets live in .env (git-ignored), generated on first use.

.DEFAULT_GOAL := help
COMPOSE := docker compose

.PHONY: help env up down reset logs ps migrate psql test test-integration security agents-sync agents-check agents-zones

help: ## List the targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-18s %s\n", $$1, $$2}'

env: ## Create .env with random local secrets (only if missing)
	@bash scripts/env.sh

up: env ## Start PostgreSQL, migrations, API, gateway and MFE (http://localhost:3000); returns when ready
	$(COMPOSE) up -d --build
	@bash scripts/wait-health.sh http://localhost:8080 90
	@bash scripts/wait-ui.sh http://localhost:$${MFE_PORT:-3000} 90

down: ## Stop the local stack (keeps the database)
	$(COMPOSE) down --remove-orphans

reset: ## Stop the local stack and delete the database volume
	$(COMPOSE) down -v --remove-orphans

logs: ## Follow API, gateway, MFE and migration logs
	$(COMPOSE) logs -f migrate api gw mfe

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

test-integration: ## Clean isolated stack + hubx-bender (Cucumber: API via gateway, @e2e in browser), then tear down
	@bash scripts/test-integration.sh

security: ## gosec and govulncheck on both Go services
	cd hubx-api && gosec ./...
	cd hubx-gw && gosec ./...
	govulncheck -C hubx-api ./...
	govulncheck -C hubx-gw ./...

agents-sync: ## Generate the Kiro and Claude Code agents from .opencode/agents
	@node scripts/sync-agents.mjs

agents-check: ## Fail if the Kiro or Claude Code agents are out of date with .opencode/agents
	@node scripts/sync-agents.mjs --check

agents-zones: ## Fail if a commit tagged [agent:<role>] touches files outside that role's zone (base..HEAD)
	@node scripts/check-agent-zones.mjs $(BASE)
