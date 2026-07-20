BINARY  := prism
VERSION := 0.1.0

.PHONY: build test cover lint clean install

build:
	go build -o $(BINARY) ./cmd/prism

install:
	go install ./cmd/prism

test:
	go test -v -race ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:
	go vet ./...

clean:
	rm -f $(BINARY) coverage.out coverage.html
