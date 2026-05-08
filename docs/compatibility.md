# Compatibility

The target baseline is Electron 42.0.0 with Chromium 148.0.7778.96, Node.js
24.15.0, and V8 14.8.178.14.

Compatibility state lives in `internal/compat/ledger.json`.

Statuses:

- `unstarted`: no implementation exists.
- `stubbed`: an explicit boundary exists, but behavior is not implemented.
- `partial`: some behavior exists, with known gaps.
- `compatible`: behavior matches the target Electron baseline with evidence.

Run:

```sh
go run ./cmd/electron-go --compat-json
go run ./cmd/electron-go --check-parity
```

`--check-parity` must fail until every ledger item is `compatible`. That failure
is intentional. It prevents accidental release notes, README text, or marketing
copy from claiming complete Electron support too early.

## Updating The Baseline

Before a release:

1. check the latest stable release at `https://releases.electronjs.org/`;
2. update the Electron, Chromium, Node.js, and V8 versions in the ledger;
3. import or adapt upstream Electron tests for changed APIs;
4. add new ledger entries for breaking changes or new public behavior;
5. rerun conformance and benchmark suites.
