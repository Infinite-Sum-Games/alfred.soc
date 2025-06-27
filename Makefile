build:
	@go fmt ./...
	@go build -o bin/alfred

run: build
	@./bin/alfred

# For docker users
docker:
	@docker compose up -d

# For podman users
dev:
	@podman compose down
	@podman compose up -d
