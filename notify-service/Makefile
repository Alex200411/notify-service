.PHONY: build test run-server run-worker migrate docker-up docker-down clean

build:
	go build -o bin/server ./cmd/server
	go build -o bin/worker ./cmd/worker

test:
	go test ./... -v -count=1

run-server:
	go run ./cmd/server

run-worker:
	go run ./cmd/worker

migrate:
	go run ./cmd/server -migrate

docker-up:
	docker compose up -d

docker-down:
	docker compose down

clean:
	rm -rf bin/
