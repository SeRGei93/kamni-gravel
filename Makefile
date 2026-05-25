.PHONY: help build run-bot run-api migrate-up migrate-down migrate-status test clean docker-up docker-down docker-logs docker-prod-build docker-prod-up docker-prod-down docker-prod-logs ssl-cert ssl-renew db-psql

COMPOSE ?= docker-compose
PROD_COMPOSE = $(COMPOSE) -f docker-compose.yml -f docker-compose.prod.yml

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build all binaries
	cd backend && go build -o ../bin/bot ./cmd/bot
	cd backend && go build -o ../bin/api ./cmd/api
	cd backend && go build -o ../bin/migrate ./cmd/migrate

run-bot: ## Run telegram bot (requires PostgreSQL running)
	cd backend && go run ./cmd/bot

run-api: ## Run REST API server (requires PostgreSQL running)
	cd backend && go run ./cmd/api

migrate-up: ## Run database migrations up (requires PostgreSQL running)
	cd backend && go run ./cmd/migrate/main.go up

migrate-down: ## Rollback database migrations (requires PostgreSQL running)
	cd backend && go run ./cmd/migrate/main.go down

migrate-status: ## Show migration status (requires PostgreSQL running)
	cd backend && go run ./cmd/migrate/main.go status

migrate-create: ## Create new migration (usage: make migrate-create NAME=create_users)
	cd backend && goose -dir internal/infrastructure/migrations create $(NAME) sql

test: ## Run tests
	cd backend && go test -v ./...

clean: ## Clean build artifacts
	rm -rf bin/

# Docker commands
docker-up: ## Start all services with docker-compose
	$(COMPOSE) up -d

docker-down: ## Stop all services
	$(COMPOSE) down

docker-down-v: ## Stop all services and remove volumes
	$(COMPOSE) down -v

docker-build: ## Build docker images
	$(COMPOSE) build

docker-logs: ## Show docker logs
	$(COMPOSE) logs -f

docker-restart: ## Restart all services
	$(COMPOSE) restart

docker-prod-build: ## Build production images with production env
	$(PROD_COMPOSE) build

docker-prod-up: ## Start production services with nginx
	$(PROD_COMPOSE) up -d --build

docker-prod-down: ## Stop production services
	$(PROD_COMPOSE) down

docker-prod-logs: ## Show production logs
	$(PROD_COMPOSE) logs -f

ssl-cert: ## Issue initial Let's Encrypt certificate for PUBLIC_DOMAIN
	./scripts/generate-ssl-cert.sh issue

ssl-renew: ## Renew Let's Encrypt certificate for PUBLIC_DOMAIN
	./scripts/generate-ssl-cert.sh renew

# Database commands
db-psql: ## Connect to PostgreSQL via psql (requires docker-compose running)
	$(COMPOSE) exec postgres psql -U gravel -d gravel_bot

db-backup: ## Backup database to ./backup/gravel_bot_backup.sql
	mkdir -p backup
	$(COMPOSE) exec -T postgres pg_dump -U gravel gravel_bot > backup/gravel_bot_backup_$(shell date +%Y%m%d_%H%M%S).sql

db-restore: ## Restore database from backup (usage: make db-restore FILE=backup/file.sql)
	$(COMPOSE) exec -T postgres psql -U gravel gravel_bot < $(FILE)

# Development
dev-deps: ## Install development dependencies
	cd backend && go mod download
	cd frontend && npm install

dev-frontend: ## Run frontend in development mode
	cd frontend && npm run dev

dev-backend: ## Run backend services in development mode
	make docker-up
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5
	make migrate-up
	make run-bot
