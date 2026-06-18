SHELL := /bin/sh

.PHONY: run run-bot run-worker test lint docker-up docker-down migrate-up migrate-down

run:
	cd backend && go run ./cmd/api

run-bot:
	cd backend && go run ./cmd/bot

run-worker:
	cd backend && go run ./cmd/worker

test:
	cd backend && go test ./...

lint:
	cd backend && gofmt -w $$(find . -name '*.go') && go test ./...

docker-up:
	docker compose -f infra/docker-compose.yml up --build

docker-down:
	docker compose -f infra/docker-compose.yml down

migrate-up:
	cd backend && migrate -path migrations -database "$$DATABASE_URL" up

migrate-down:
	cd backend && migrate -path migrations -database "$$DATABASE_URL" down
