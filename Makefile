.PHONY: up down run seed swag test

up:
	docker compose up -d --build

down:
	docker compose down

run:
	go run ./cmd/api

seed:
	go run ./cmd/seed

swag:
	swag init -g cmd/api/main.go -o docs

test:
	go test ./...
