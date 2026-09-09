.PHONY: test run build docker-up docker-down

test:
	go test ./...

run:
	go run .

build:
	go build -o bin/mana .

docker-up:
	docker compose up --build

docker-down:
	docker compose down

