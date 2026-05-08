# CEF Bootstrap

`cef_bootstrap` is the first packet that can move the parity score from `0/45`.
It is intentionally narrower than Electron compatibility:

- Linux x86_64 first;
- one visible `BrowserWindow`;
- one `file://` load from the hello fixture;
- clean exit on window close;
- no zombie renderer/GPU/network subprocesses;
- no JavaScript bridge or Electron IPC.

## Target

Electron 42.0.0 uses Chromium 148.0.7778.96. The CEF artifact must target
Chromium major `148`; Chromium 130 is not acceptable for this baseline.

The target is recorded in `native/CEF_VERSION`. Resolve it with:

```sh
go run ./tools/cefresolve --json
```

The command exits nonzero if no exact `linux64` `minimal` CEF artifact exists for
Chromium 148.0.7778.96. That failure is useful: it blocks accidental linkage to
the wrong Chromium major.

For architecture-only experiments, the resolver can show the newest available
same-major artifact:

```sh
go run ./tools/cefresolve --json --allow-major
```

Do not mark `cef_bootstrap` compatible from a same-major fallback. Compatibility
evidence must record the exact Chromium/Electron baseline used.

Once resolution succeeds, fetch and extract the archive deterministically:

```sh
go run ./tools/ceffetch --json --output native/cef
```

`ceffetch` verifies the CEF archive SHA-1 from the Spotify build index before it
extracts.

## Runtime Layout

CEF runtime files must sit next to the executable:

```text
bin/
  electron-go
  libcef.so
  icudtl.dat
  v8_context_snapshot.bin
  locales/
    en-US.pak
  chrome_100_percent.pak
  resources.pak
```

Validate before running:

```sh
go run ./tools/memaudit --root . --check-cef-layout ./bin
```

## cgo Linkage

The Linux CEF build must provide:

```sh
CGO_CFLAGS="-I${CEF_ROOT}/include"
CGO_LDFLAGS="-L${CEF_ROOT}/Release -lcef -Wl,-rpath,\$ORIGIN"
```

The `$ORIGIN` rpath is required so `libcef.so` can be resolved beside
`bin/electron-go` in CI and packaged builds.

## Shim Rule

CEF callback/vtable structs are populated in C, not Go. Go can own the logic, but
the C shim owns function-pointer storage. The shim sources live in
`native/cef_shim.c` and `native/cef_shim.h`.
