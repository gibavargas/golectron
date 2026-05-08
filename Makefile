.PHONY: test bench build run-hello parity

test:
	go test ./...

bench:
	go test -run '^$$' -bench . -benchmem ./...

build:
	go build -o bin/electron-go ./cmd/electron-go

run-hello:
	go run ./cmd/electron-go ./compat/fixtures/hello

parity:
	go run ./cmd/electron-go --check-parity
