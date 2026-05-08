.PHONY: test bench build run-hello parity workpackets cef-packet memaudit releasecheck conformance

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

workpackets:
	go run ./tools/workpackets --status unstarted,stubbed,partial --limit 10

cef-packet:
	go run ./tools/workpackets --id cef_bootstrap

memaudit:
	go run ./tools/memaudit --root .

releasecheck:
	go run ./tools/releasecheck --json

conformance:
	go run ./tools/conformance --fixture ./compat/fixtures/hello --electron electron --electron-go "go run ./cmd/electron-go" --allow-mismatch
