.PHONY: all build test run clean

BINARY := bin/syshud
CMD := ./cmd/hud

all: build

build:
	@mkdir -p bin
	go build -ldflags="-s -w" -o $(BINARY) $(CMD)

test:
	go test -v -bench=. -benchmem ./...

run: build
	./$(BINARY)

clean:
	rm -rf bin

