build:
	go build -o ./.bin cmd/proxy/main.go

run: build
	./.bin