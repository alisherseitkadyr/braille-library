.PHONY: up down run seed swag test

up:
	docker compose up -d --build

down:
	docker compose down

run:
	go run ./cmd/api

# make seed ARGS=--reset to replace existing data
seed:
	go run ./cmd/seed $(ARGS)

swag:
	swag init -g cmd/api/main.go -o docs

test:
	go test ./...
