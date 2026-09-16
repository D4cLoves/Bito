.PHONY: help up down logs test test-race lint build run

help: ## Show help for each of the Makefile recipes.
	@echo "Available commands:"
	@echo "  make up         - Start docker infrastructure (Postgres, Redis, Mailpit)"
	@echo "  make down       - Stop docker infrastructure"
	@echo "  make logs       - View docker logs"
	@echo "  make test       - Run all unit tests"
	@echo "  make test-race  - Run unit tests with Go data race detector"
	@echo "  make run        - Run backend server locally"
	@echo "  make build      - Build server binary"

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

test:
	go test -v ./...

test-race:
	go test -v -race ./...

build:
	go build -o bin/server.exe cmd/server/main.go

run:
	go run cmd/server/main.go
