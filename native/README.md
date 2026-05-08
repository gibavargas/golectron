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

## Platform skeleton

`internal/native` exposes `NewBridge()` behind build tags for Darwin, Linux,
Windows, and other platforms. Every implementation currently returns
`StubBridge`, which validates requests and then reports
`ErrBridgeUnavailable`. This is deliberate: parity must not be claimed until
the bundled Chromium/Node/V8 boot path is present.

The first bridge target is macOS Apple Silicon. Linux and Windows must be added
before full parity can be claimed.
