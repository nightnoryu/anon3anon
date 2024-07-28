all: build

.PHONY: build
build: modules
	CGO_ENABLED=0 go build -o ./bin/anon3anon ./cmd/anon3anon

.PHONY: modules
modules:
	go mod tidy
