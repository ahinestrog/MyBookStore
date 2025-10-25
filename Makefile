SHELL := /bin/bash

.PHONY: up down restart logs ps nuke gen

gen:
	@bash scripts/gen_stubs.sh || true

up:
	@docker compose pull --ignore-pull-failures || true
	@docker compose build
	@docker compose up -d
	@docker compose ps

down:
	@docker compose down

restart:
	@docker compose down
	@docker compose up -d

logs:
	@docker compose logs -f --tail=200

ps:
	@docker compose ps

# Elimina todos los contenedores
nuke:
	@docker compose down -v --remove-orphans
