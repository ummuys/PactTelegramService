
# Deploy

COMPOSE = docker compose
COMPOSE_FILE = docker-compose.yaml

.PHONY: up down code-review code-auto-format

up:
	$(COMPOSE) -f $(COMPOSE_FILE) up -d --build

down:
	$(COMPOSE) -f $(COMPOSE_FILE) down


# CI

code-review:
	@cd .. && ./deployment/scripts/pre_commit.sh

code-auto-format:
	go install mvdan.cc/gofumpt@latest
	@cd .. && gofumpt -w $$(find . -name '*.go' -type f)