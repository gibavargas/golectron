# Compatibility

The target baseline is Electron 42.0.0 with Chromium 148.0.7778.96, Node.js
24.15.0, and V8 14.8.178.14.

Compatibility state lives in `internal/compat/ledger.json`. The ledger is an
implementation inventory, not a marketing checklist: entries may cite upstream
Electron requirements, release notes, local implementation paths, docs, or tests,
but only entries with real local implementation and conformance evidence may be
marked `compatible`.

Statuses:

- `unstarted`: no implementation exists.
- `stubbed`: an explicit boundary exists, but behavior is not implemented.
- `partial`: some behavior exists, with known gaps.
- `compatible`: behavior matches the target Electron baseline with evidence.

Every item must include a nonempty `id`, `area`, `api_group`, `status`,
`evidence`, and `notes`. Evidence is typed:

- `release-note`: a version-specific Electron requirement or behavior change.
- `upstream`: upstream Electron API documentation or baseline metadata.
- `implementation`: local code that implements some or all of the behavior.
- `test` or `conformance`: local tests that exercise the behavior.
- `documentation`: local docs that describe constraints or current gaps.

`compatible` is intentionally strict: it requires implementation evidence and
should also have conformance evidence for the behavior being claimed. Release
notes and upstream docs alone are inventory evidence, not compatibility proof.

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
3. add release-note entries for breaking changes, new APIs, and behavior fixes
   that affect app-visible behavior;
4. import or adapt upstream Electron tests for changed APIs;
5. add local implementation and conformance evidence as work lands;
6. rerun conformance and benchmark suites.
