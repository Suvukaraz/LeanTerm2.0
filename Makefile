.PHONY: all build test lint clean

all: build

build:
	go build -o bin/pureterm ./cmd/pureterm

test:
	go test -race ./...

lint:
	go vet ./...

clean:
	rm -rf bin/
