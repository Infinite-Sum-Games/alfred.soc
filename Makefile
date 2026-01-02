build:
	@go fmt ./...
	@go build -o bin/alfred

run: build
	@./bin/alfred

# Ngrok startup. Change this to your unqiue NGROK domain from the dashboard
grok:
	@ngrok http 9001 --domain unique-pure-flamingo.ngrok-free.app
