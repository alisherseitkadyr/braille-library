.PHONY: up down run seed recompute swag test

up:
	docker compose up -d --build

down:
	docker compose down

run:
	go run ./cmd/api

# make seed ARGS=--reset to replace existing data
seed:
	go run ./cmd/seed $(ARGS)

recompute:
	go run ./cmd/recompute

swag:
	swag init -g cmd/api/main.go -o docs --parseInternal

test:
	go test ./...
