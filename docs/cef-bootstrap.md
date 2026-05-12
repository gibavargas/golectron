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

Once resolution succeeds, fetch, verify, extract, and stage the runtime layout
deterministically:

```sh
tools/fetch_cef.sh
```

`ceffetch` verifies the CEF archive SHA-1 from the Spotify build index before it
extracts, and can also enforce a `SHA256` value recorded in `native/CEF_VERSION`
when the build page publishes one for the exact tarball.

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

The fetcher stages all Linux runtime `.so` companions, resource packs, snapshot
files, and locale packs it finds. It also creates `native/cef/current` so the
Linux cgo build can include headers from a stable path without vendoring CEF.

## cgo Linkage

The Linux CEF-linked build is enabled explicitly with the `electron_go_cef` tag:

```sh
go build -tags electron_go_cef -o bin/electron-go ./cmd/electron-go
```

The Linux cgo directives live in `internal/native/cef_linux.go` and bake in:

```text
-I${SRCDIR}/../../native/cef/current
-L${SRCDIR}/../../bin -lcef -Wl,-rpath,$ORIGIN
```

The `$ORIGIN` rpath is required so `libcef.so` can be resolved beside
`bin/electron-go` in CI and packaged builds. The default Linux build remains the
stub bridge until the tag is provided.

## Ubuntu 22.04 CI

The bootstrap workflow runs on Ubuntu 22.04 Jammy, not Alpine. The prebuilt CEF
artifacts expect glibc and X11/GTK libraries, and the headless acceptance run
uses Xvfb:

```sh
sudo apt-get install -y libgtk-3-0 libgdk-pixbuf2.0-0 libglib2.0-0 \
  libnss3 libnspr4 libatk1.0-0 libatk-bridge2.0-0 libcups2 libdrm2 \
  libxkbcommon0 libxcomposite1 libxdamage1 libxfixes3 libxrandr2 \
  libgbm1 libasound2 xvfb
```

`no_sandbox = 1` is acceptable only for this bootstrap packet and must remain
visible as a compatibility gap until sandboxed renderer behavior has parity
evidence.

## Shim Rule

CEF callback/vtable structs are populated in C, not Go. Go can own the logic, but
the C shim owns function-pointer storage. The shim sources live in
`native/cef_shim.c` and `native/cef_shim.h`.
