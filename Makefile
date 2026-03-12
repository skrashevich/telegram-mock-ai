.PHONY: build run clean test

BINARY=telegram-mock-ai

build:
	go build -o $(BINARY) ./cmd/telegram-mock-ai

run: build
	./$(BINARY) -config config.example.yaml

test:
	go test ./... -v

clean:
	rm -f $(BINARY)
