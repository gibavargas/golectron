# Native Bridge

Electron-Go uses Go as the runtime host and a native bridge for bundled
Chromium, Node.js, and V8.

The bridge contract starts narrow on purpose:

1. Go loads app metadata and owns lifecycle state.
2. Go passes a validated start request to native code.
3. Native code boots Chromium, Node.js, V8, and Electron-compatible JS bindings.
4. Native code returns observable process/window/runtime details for tests and
   benchmarks.

The first bridge target is macOS Apple Silicon. Linux and Windows must be added
before full parity can be claimed.
