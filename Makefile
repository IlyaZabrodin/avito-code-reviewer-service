.PHONY: build run test clean docker-build docker-up docker-down docker-logs docker-restart migrate-up migrate-down deps

build:
	go mod tidy
	mkdir -p bin
	go clean -cache
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/code_review_service ./cmd/main.go

run:
	go run ./cmd/main.go

test:
	go test -v ./...

clean:
	rm -rf bin
	go clean -cache

docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f app

docker-restart: docker-down docker-up

migrate-up:
	docker-compose exec app ./pr-reviewer migrate up

migrate-down:
	docker-compose exec app ./pr-reviewer migrate down

deps:
	go mod download
