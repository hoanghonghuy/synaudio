.PHONY: up down logs migrate-up migrate-down sqlc test backend-test frontend-install frontend-dev api worker tidy

up:
	docker compose up -d postgres minio minio-init

down:
	docker compose down

logs:
	docker compose logs -f postgres minio

migrate-up:
	docker run --rm --network host -v "$(CURDIR)/backend/db/migrations:/migrations" migrate/migrate:v4.18.3 -path=/migrations -database "$$DATABASE_URL" up

migrate-down:
	docker run --rm --network host -v "$(CURDIR)/backend/db/migrations:/migrations" migrate/migrate:v4.18.3 -path=/migrations -database "$$DATABASE_URL" down 1

sqlc:
	docker run --rm -v "$(CURDIR)/backend:/src" -w /src/db sqlc/sqlc:1.29.0 generate

backend-test:
	cd backend && go test ./...

test: backend-test

tidy:
	cd backend && go mod tidy

api:
	cd backend && go run ./cmd/api

worker:
	cd backend && go run ./cmd/worker

frontend-install:
	cd frontend && npm install

frontend-dev:
	cd frontend && npm run dev
