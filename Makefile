.PHONY: test bench build run-hello parity workpackets cef-packet cefresolve ceffetch fetch-cef cef-layout memaudit releasecheck conformance local-ci local-goal-audit local-benchmarks

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

cefresolve:
	go run ./tools/cefresolve --json

ceffetch:
	go run ./tools/ceffetch --json

fetch-cef:
	tools/fetch_cef.sh

cef-layout:
	go run ./tools/memaudit --root . --check-cef-layout ./bin

memaudit:
	go run ./tools/memaudit --root .

releasecheck:
	go run ./tools/releasecheck --json

conformance:
	go run ./tools/conformance --fixture ./compat/fixtures/hello --electron electron --electron-go "go run ./cmd/electron-go" --allow-mismatch

local-ci:
	tools/local_actions.sh ci

local-goal-audit:
	tools/local_actions.sh goal-audit

local-benchmarks:
	tools/local_actions.sh benchmarks
