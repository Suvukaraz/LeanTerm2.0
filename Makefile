.PHONY: all build test lint clean

all: build

build:
	go build -o bin/leanterm ./cmd/leanterm

test:
	go test -race ./...

lint:
	go vet ./...

clean:
	rm -rf bin/
