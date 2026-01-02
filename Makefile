build:
	@go fmt ./...
	@go build -o bin/alfred

run: build
	@./bin/alfred

# Ngrok startup
grok:
	@ngrok http 9001 --domain unique-pure-flamingo.ngrok-free.app
