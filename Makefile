all: build

build: modules
	CGO_ENABLED=0 go build -o ./bin/anon3anon ./cmd/anon3anon

modules:
	go mod tidy

test:
	go test ./..
