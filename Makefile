build:
	@go fmt ./...
	@go build -o bin/alfred

run: build
	@./bin/alfred

dev:
	@docker-compose up -d
