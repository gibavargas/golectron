# Native Bridge

Electron-Go uses Go as the runtime host and a native bridge for bundled
Chromium, Node.js, and V8. The bridge is intentionally unavailable today: the
Go runtime validates the launch request and the platform bridge returns a clear
unavailable error until a real Chromium/Node/V8 integration is linked.

## ABI status

- Current ABI revision: `1`
- Public C header: `electron_go_bridge.h`
- Go contract: `internal/native`
- Implemented engines: none
- Expected status today: `EG_BRIDGE_STATUS_UNAVAILABLE`
- CEF bootstrap status: deterministic fetch/link scaffolding only; no linked CEF
  runtime is booted by default.

## Launch contract

The bridge contract starts narrow on purpose:

1. Go loads app metadata and owns lifecycle state.
2. Go builds and validates a start request with app paths, app identity, target
   Electron version, argv, and environment data.
3. Native code boots bundled Chromium, Node.js, V8, and Electron-compatible JS
   bindings.
4. Native code returns observable process/window/runtime details for tests and
   benchmarks.

Request strings use pointer-plus-length `eg_string_view` values so native code
does not depend on NUL termination. Memory returned in `eg_bridge_start_result`
belongs to the bridge and must be released with `eg_bridge_free_result`.

## CEF bootstrap ABI

The CEF-facing ABI is split into process, initialization, browser, message-loop,
and shutdown calls:

1. `eg_cef_execute_process` receives subprocess argv and returns
   `eg_cef_subprocess_result` with the CEF subprocess exit code, status, and
   optional error message.
2. `eg_cef_initialize` receives `eg_cef_initialize_request`, including
   `eg_cef_settings`.
3. `eg_cef_create_browser_sync` receives
   `eg_browser_window_create_request` with an initial URL, dimensions, and
   visibility flag, and returns a browser id in `eg_browser_window_result`.
4. `eg_cef_load_url` receives `eg_browser_window_load_request` for a known
   browser id.
5. `eg_cef_run_message_loop` owns the CEF message loop until shutdown.
6. `eg_cef_shutdown` tears the bridge down.

The settings currently exposed are deliberately narrow:

- `no_sandbox`: maps to CEF sandbox disabling when the embedding platform
  cannot launch with CEF's sandbox support.
- `cache_path`: required persistent cache/user-data location.
- `log_severity`: one of default, verbose, info, warning, error, fatal, or
  disable.

All CEF lifecycle calls must be made from the process main thread. The Go host
must arrange thread ownership before crossing the C ABI, typically by locking
the OS thread around bootstrap and message-loop ownership. Native code must not
call back into Go from arbitrary CEF threads unless the callback has an explicit
threading contract.

Callbacks crossing the C/Go boundary must be treated as borrowed, synchronous
calls. Native code must not retain Go pointers, Go-owned buffers, or callback
closures after the callback returns. Callback bodies must avoid blocking the CEF
UI thread and must copy any data they need to keep.

## Linux notes

Linux CEF embedding has extra process and sandbox constraints:

- Subprocess routing must happen early enough that CEF helper processes can
  exit through `eg_cef_execute_process` without entering the Go app bootstrap.
- Signal handling must be coordinated with Go's runtime signal handlers. The
  native bridge should document any signal handlers CEF installs and avoid
  overriding Go-owned handlers after initialization.
- CEF sandbox support may require a setuid sandbox helper, user namespaces, or
  running with `no_sandbox` in constrained environments. `no_sandbox` is an ABI
  setting for bootstrapping only, not a parity claim.

## Platform skeleton

`internal/native` exposes `NewBridge()` behind build tags for Darwin, Linux,
Windows, and other platforms. The default implementations return `StubBridge`,
which validates requests and then reports `ErrBridgeUnavailable`. Linux x86_64
also has an opt-in `electron_go_cef` cgo build that links against `bin/libcef.so`
and includes headers from `native/cef/current`; it still reports unavailable
until the real CEF initialize/create-window/message-loop calls are implemented.
This is deliberate: parity must not be claimed until the bundled
Chromium/Node/V8 boot path is present.

The first bridge target is macOS Apple Silicon. Linux and Windows must be added
before full parity can be claimed.
