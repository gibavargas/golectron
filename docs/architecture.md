# Architecture

Electron-Go is split into three layers.

## Go Host

The Go host owns deterministic orchestration:

- app metadata loading;
- lifecycle state;
- IPC routing;
- permissions and policy;
- conformance and benchmark instrumentation;
- process supervision.

## Native Runtime Bridge

The native bridge owns bundled Chromium, Node.js, and V8. It must expose a small
ABI to Go while preserving Electron behavior above it. The bridge is allowed to
use C++ internally because Chromium, Node.js, and V8 are native C++ projects.

The first browser implementation target is CEF bootstrap. That target exists to
prove the non-negotiable ownership model before API parity work builds on it:

- the process-original OS main thread is locked at executable entry;
- CEF subprocess execution is checked before normal app startup;
- CEF initialization and message-loop ownership stay behind the native ABI;
- callbacks into Go copy data immediately and never store Go heap pointers in C.

## JS Compatibility Layer

The JS layer must expose Electron-compatible modules to application code. The
compatibility ledger is the release gate: APIs are not done because they exist;
they are done only when they match Electron behavior with evidence.

## Platform Order

1. macOS Apple Silicon
2. Linux x64/arm64
3. Windows x64/arm64

Full parity requires all supported platforms to pass the conformance suite.
