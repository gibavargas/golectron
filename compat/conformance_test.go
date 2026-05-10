package compat_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestHelloFixtureConformance(t *testing.T) {
	if os.Getenv("ELECTRON_GO_ENABLE_IPC_CONFORMANCE") != "1" {
		t.Skip("set ELECTRON_GO_ENABLE_IPC_CONFORMANCE=1 after IPC/preload parity lands")
	}

	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	fixture := filepath.Join("fixtures", "hello")
	electron := runFixture(t, electronBin, fixture)
	electronGo := runFixture(t, electronGoBin, fixture)

	if electron.ExitCode != electronGo.ExitCode {
		t.Fatalf("exit code mismatch: electron=%d electron-go=%d\nElectron-Go output:\n%s", electron.ExitCode, electronGo.ExitCode, electronGo.Output)
	}
	if !strings.Contains(electronGo.Output, "pong") && strings.Contains(electron.Output, "pong") {
		t.Fatalf("Electron-Go did not produce fixture IPC result found in Electron output")
	}
}

func TestBenchmarkHelloFixtureConformance(t *testing.T) {
	if os.Getenv("ELECTRON_GO_ENABLE_RUNTIME_FIXTURE_CONFORMANCE") != "1" {
		t.Skip("set ELECTRON_GO_ENABLE_RUNTIME_FIXTURE_CONFORMANCE=1 after native Chromium/Node/V8 bridge parity lands")
	}

	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	fixture := filepath.Join("fixtures", "benchmark-hello")
	electron := runFixture(t, electronBin, fixture)
	electronGo := runFixture(t, electronGoBin, fixture)

	if electron.ExitCode != 0 {
		t.Fatalf("official Electron fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go fixture exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}
}

func TestClipboardTextConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "clipboard-text"))
	electronGo := runCommand(t, electronGoBin, "--clipboard-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron clipboard fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go clipboard check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseClipboard(t, electron.Output)
	electronGoReport := parseClipboard(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("clipboard report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestClipboardFormatsConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "clipboard-formats"))
	electronGo := runCommand(t, electronGoBin, "--clipboard-formats-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron clipboard formats fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go clipboard formats check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseClipboardFormats(t, electron.Output)
	electronGoReport := parseClipboardFormats(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("clipboard formats report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestShellOpenPathConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "shell-open-path"))
	electronGo := runCommand(t, electronGoBin, "--shell-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron shell fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go shell check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseShell(t, electron.Output)
	electronGoReport := parseShell(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("shell report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestContentTracingConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "content-tracing"))
	electronGo := runCommand(t, electronGoBin, "--content-tracing-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron contentTracing fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go contentTracing check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseContentTracing(t, electron.Output)
	electronGoReport := parseContentTracing(t, electronGo.Output)
	if !reflect.DeepEqual(electronReport, electronGoReport) {
		t.Fatalf("contentTracing report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestCrashReporterConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "crash-reporter"))
	electronGo := runCommand(t, electronGoBin, "--crash-reporter-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron crashReporter fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go crashReporter check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseCrashReporter(t, electron.Output)
	electronGoReport := parseCrashReporter(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("crashReporter report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestBrowserWindowOptionsConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "browser-window-options"))
	electronGo := runCommand(t, electronGoBin, "--browser-window-options-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron BrowserWindow fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go BrowserWindow options check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseBrowserWindowOptions(t, electron.Output)
	electronGoReport := parseBrowserWindowOptions(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("BrowserWindow options report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestBrowserWindowMethodsConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "browser-window-methods"))
	electronGo := runCommand(t, electronGoBin, "--browser-window-methods-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron BrowserWindow methods fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go BrowserWindow methods check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseBrowserWindowMethods(t, electron.Output)
	electronGoReport := parseBrowserWindowMethods(t, electronGo.Output)
	// AppKit may clamp the top-level window Y coordinate to account for the
	// menu bar/titlebar on the active display. The method subset validated here
	// is visibility, sizing, normal-bounds dimensions, close/closed events, and
	// enabled state, so normalize the host-dependent Y coordinate before the
	// structural comparison.
	electronReport.Bounds.Y = electronGoReport.Bounds.Y
	electronReport.NormalBounds.Y = electronGoReport.NormalBounds.Y
	if !reflect.DeepEqual(electronReport, electronGoReport) {
		t.Fatalf("BrowserWindow methods report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestViewTreeConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "view-tree"))
	electronGo := runCommand(t, electronGoBin, "--view-tree-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron View tree fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go View tree check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseViewTree(t, electron.Output)
	electronGoReport := parseViewTree(t, electronGo.Output)
	if !equalIntSlices(electronReport.RootChildWidths, electronGoReport.RootChildWidths) ||
		!equalIntSlices(electronReport.AfterRemoveWidths, electronGoReport.AfterRemoveWidths) ||
		!equalIntSlices(electronReport.AfterReparentWidths, electronGoReport.AfterReparentWidths) ||
		!equalIntSlices(electronReport.NewParentChildWidths, electronGoReport.NewParentChildWidths) ||
		electronReport.BoundsChanged != electronGoReport.BoundsChanged ||
		electronReport.ProbeBounds != electronGoReport.ProbeBounds {
		t.Fatalf("View tree report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestViewAnimationConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "view-animation"))
	electronGo := runCommand(t, electronGoBin, "--view-animation-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron View animation fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go View animation check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseViewAnimation(t, electron.Output)
	electronGoReport := parseViewAnimation(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("View animation report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestWebContentsNavigationConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "webcontents-navigation"))
	electronGo := runCommand(t, electronGoBin, "--webcontents-navigation-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron webContents navigation fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go webContents navigation check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseWebContentsNavigation(t, electron.Output)
	electronGoReport := parseWebContentsNavigation(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("webContents navigation report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestWebContentsDevToolsTargetConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "webcontents-devtools-target"))
	electronGo := runCommand(t, electronGoBin, "--webcontents-devtools-target-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron webContents DevTools target fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go webContents DevTools target check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseWebContentsDevToolsTarget(t, electron.Output)
	electronGoReport := parseWebContentsDevToolsTarget(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("webContents DevTools target report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestWebContentsPrintDefaultPageSizeConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "webcontents-print"))
	electronGo := runCommand(t, electronGoBin, "--webcontents-print-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron webContents print fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go webContents print check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseWebContentsPrint(t, electron.Output)
	electronGoReport := parseWebContentsPrint(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("webContents print report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestFocusOnNavigationConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "focus-on-navigation"))
	electronGo := runCommand(t, electronGoBin, "--focus-on-navigation-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron focusOnNavigation fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go focusOnNavigation check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseFocusOnNavigation(t, electron.Output)
	electronGoReport := parseFocusOnNavigation(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("focusOnNavigation report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestOffscreenDeviceScaleConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "offscreen-device-scale"))
	electronGo := runCommand(t, electronGoBin, "--offscreen-device-scale-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron offscreen device scale fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go offscreen device scale check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseOffscreenDeviceScale(t, electron.Output)
	electronGoReport := parseOffscreenDeviceScale(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("offscreen device scale report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestIPCInvokeConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "ipc-invoke"))
	electronGo := runCommand(t, electronGoBin, "--ipc-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron IPC fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go IPC check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseIPC(t, electron.Output)
	electronGoReport := parseIPC(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("IPC report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestContextBridgeConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "context-bridge"))
	electronGo := runCommand(t, electronGoBin, "--context-bridge-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron contextBridge fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go contextBridge check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseContextBridge(t, electron.Output)
	electronGoReport := parseContextBridge(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("contextBridge report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestAppIsActiveConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "app-is-active"))
	electronGo := runCommand(t, electronGoBin, "--app-is-active-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron app.isActive fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go app.isActive check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseAppActivity(t, electron.Output)
	electronGoReport := parseAppActivity(t, electronGo.Output)
	if electronReport.Platform != electronGoReport.Platform {
		t.Fatalf("platform mismatch: electron=%s electron-go=%s", electronReport.Platform, electronGoReport.Platform)
	}
	if electronReport.Supported != electronGoReport.Supported {
		t.Fatalf("supported mismatch on %s: electron=%v electron-go=%v", electronReport.Platform, electronReport.Supported, electronGoReport.Supported)
	}
	if !electronGoReport.Supported && electronGoReport.Active {
		t.Fatalf("Electron-Go reported active=true on unsupported platform %s", electronGoReport.Platform)
	}
}

func TestAppLifecycleConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "app-lifecycle"))
	electronGo := runCommand(t, electronGoBin, "--app-lifecycle-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron app lifecycle fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go app lifecycle check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseAppLifecycle(t, electron.Output)
	electronGoReport := parseAppLifecycle(t, electronGo.Output)
	if strings.Join(electronReport.EventOrder, ",") != strings.Join(electronGoReport.EventOrder, ",") {
		t.Fatalf("app lifecycle event order mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
	if electronReport.Ready != electronGoReport.Ready || electronReport.WhenReady != electronGoReport.WhenReady {
		t.Fatalf("app lifecycle ready mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
	if electronReport.CancelledBeforeQuit != electronGoReport.CancelledBeforeQuit {
		t.Fatalf("app lifecycle cancellation mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
	if electronReport.QuitExitCode != electronGoReport.QuitExitCode {
		t.Fatalf("app lifecycle quit exit mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestAppPathsConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "app-paths"))
	electronGo := runCommand(t, electronGoBin, "--app-paths-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron app paths fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go app paths check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseAppPaths(t, electron.Output)
	electronGoReport := parseAppPaths(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("app paths report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestAsarArchiveConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	archivePath := writeAsarFixtureArchive(t)
	electron := runFixtureWithEnv(t, electronBin, filepath.Join("fixtures", "asar-archive"), "ASAR_FIXTURE_ARCHIVE="+archivePath)
	electronGo := runCommand(t, electronGoBin, "--asar-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron ASAR fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go ASAR check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseAsar(t, electron.Output)
	electronGoReport := parseAsar(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("ASAR report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestAppWebAuthnConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "app-webauthn"))
	electronGo := runCommand(t, electronGoBin, "--app-webauthn-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron app WebAuthn fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go app WebAuthn check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseAppWebAuthn(t, electron.Output)
	electronGoReport := parseAppWebAuthn(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("app WebAuthn report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestAutoUpdaterFeedURLConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "auto-updater"))
	electronGo := runCommand(t, electronGoBin, "--auto-updater-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron autoUpdater fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go autoUpdater check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseAutoUpdater(t, electron.Output)
	electronGoReport := parseAutoUpdater(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("autoUpdater report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestNodeExperimentalTransformTypesConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runCommand(t, electronBin, "--experimental-transform-types", filepath.Join("fixtures", "node-transform-types"))
	electronGo := runCommand(t, electronGoBin, "--experimental-transform-types", "--node-options-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron transform-types fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go transform-types check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseNodeTransformTypes(t, electron.Output)
	electronGoReport := parseNodeTransformTypes(t, electronGo.Output)
	if !electronReport.Supported {
		t.Fatalf("official Electron did not report --experimental-transform-types support\nOutput:\n%s", electron.Output)
	}
	if electronReport.Supported != electronGoReport.Supported {
		t.Fatalf("supported mismatch: electron=%v electron-go=%v", electronReport.Supported, electronGoReport.Supported)
	}
	if electronReport.ExperimentalTransformTypes != electronGoReport.ExperimentalTransformTypes {
		t.Fatalf("experimentalTransformTypes mismatch: electron=%v electron-go=%v", electronReport.ExperimentalTransformTypes, electronGoReport.ExperimentalTransformTypes)
	}
	if electronReport.ExecArgvIncludesFlag != electronGoReport.ExecArgvIncludesFlag {
		t.Fatalf("execArgvIncludesFlag mismatch: electron=%v electron-go=%v", electronReport.ExecArgvIncludesFlag, electronGoReport.ExecArgvIncludesFlag)
	}
	if electronReport.Node == "" || electronGoReport.Node == "" {
		t.Fatalf("node versions must be reported: electron=%q electron-go=%q", electronReport.Node, electronGoReport.Node)
	}
}

func TestNodeVersionsConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "node-versions"))
	electronGo := runCommand(t, electronGoBin, "--node-versions-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron node versions fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go node versions check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseNodeVersions(t, electron.Output)
	electronGoReport := parseNodeVersions(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("node versions report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestNetLogLifecycleConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "netlog-lifecycle"))
	electronGo := runCommand(t, electronGoBin, "--netlog-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron netLog fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go netLog check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseNetLog(t, electron.Output)
	electronGoReport := parseNetLog(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("netLog report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestClientRequestConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "client-request"))
	electronGo := runCommand(t, electronGoBin, "--client-request-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron ClientRequest fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go ClientRequest check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseClientRequest(t, electron.Output)
	electronGoReport := parseClientRequest(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("ClientRequest report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestGlobalShortcutSuspensionConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "global-shortcut-suspension"))
	electronGo := runCommand(t, electronGoBin, "--global-shortcut-suspension-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron globalShortcut suspension fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go globalShortcut suspension check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseGlobalShortcutSuspension(t, electron.Output)
	electronGoReport := parseGlobalShortcutSuspension(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("globalShortcut suspension report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestMenuTemplateConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "menu-template"))
	electronGo := runCommand(t, electronGoBin, "--menu-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron menu fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go menu check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseMenu(t, electron.Output)
	electronGoReport := parseMenu(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("menu report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestNativeThemeConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "native-theme"))
	electronGo := runCommand(t, electronGoBin, "--native-theme-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron nativeTheme fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go nativeTheme check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseNativeTheme(t, electron.Output)
	electronGoReport := parseNativeTheme(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("nativeTheme report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestNotificationOptionsConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "notification-options"))
	electronGo := runCommand(t, electronGoBin, "--notification-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron notification fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go notification check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseNotification(t, electron.Output)
	electronGoReport := parseNotification(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("notification report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestNotificationIdentityConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "notification-identity"))
	electronGo := runCommand(t, electronGoBin, "--notification-identity-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron notification identity fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go notification identity check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseNotificationIdentity(t, electron.Output)
	electronGoReport := parseNotificationIdentity(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("notification identity report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestSafeStorageConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "safe-storage"))
	electronGo := runCommand(t, electronGoBin, "--safe-storage-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron safeStorage fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go safeStorage check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseSafeStorage(t, electron.Output)
	electronGoReport := parseSafeStorage(t, electronGo.Output)
	if electronReport.Available != electronGoReport.Available {
		t.Fatalf("availability mismatch: electron=%v electron-go=%v", electronReport.Available, electronGoReport.Available)
	}
	if electronReport.AsyncAvailable != electronGoReport.AsyncAvailable {
		t.Fatalf("async availability mismatch: electron=%v electron-go=%v", electronReport.AsyncAvailable, electronGoReport.AsyncAvailable)
	}
	if electronReport.RoundTrip != electronGoReport.RoundTrip {
		t.Fatalf("roundTrip mismatch: electron=%v electron-go=%v", electronReport.RoundTrip, electronGoReport.RoundTrip)
	}
	if electronReport.RoundTrip && (electronReport.CiphertextBytes == 0 || electronGoReport.CiphertextBytes == 0) {
		t.Fatalf("ciphertext bytes must be nonzero: electron=%d electron-go=%d", electronReport.CiphertextBytes, electronGoReport.CiphertextBytes)
	}
}

func TestProtocolRegistrationConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "protocol-registration"))
	electronGo := runCommand(t, electronGoBin, "--protocol-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron protocol fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go protocol check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseProtocol(t, electron.Output)
	electronGoReport := parseProtocol(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("protocol report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestSessionBasicConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "session-basic"))
	electronGo := runCommand(t, electronGoBin, "--session-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron session fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go session check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseSession(t, electron.Output)
	electronGoReport := parseSession(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("session report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestSessionClearStorageQuotasConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "session-quotas"))
	electronGo := runCommand(t, electronGoBin, "--session-quotas-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron session quotas fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go session quotas check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseSessionQuotas(t, electron.Output)
	electronGoReport := parseSessionQuotas(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("session quotas report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestSessionWebAuthnSelectionConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "session-webauthn"))
	electronGo := runCommand(t, electronGoBin, "--session-webauthn-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron session WebAuthn fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go session WebAuthn check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseSessionWebAuthn(t, electron.Output)
	electronGoReport := parseSessionWebAuthn(t, electronGo.Output)
	if electronReport != electronGoReport {
		t.Fatalf("session WebAuthn report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

func TestUtilityProcessConformance(t *testing.T) {
	electronBin := os.Getenv("ELECTRON_BIN")
	electronGoBin := os.Getenv("ELECTRON_GO_BIN")
	if electronBin == "" || electronGoBin == "" {
		t.Skip("set ELECTRON_BIN and ELECTRON_GO_BIN to run official Electron vs Electron-Go conformance")
	}

	electron := runFixture(t, electronBin, filepath.Join("fixtures", "utility-process"))
	electronGo := runCommand(t, electronGoBin, "--utility-process-check")
	if electron.ExitCode != 0 {
		t.Fatalf("official Electron utilityProcess fixture exit code = %d\nOutput:\n%s", electron.ExitCode, electron.Output)
	}
	if electronGo.ExitCode != 0 {
		t.Fatalf("Electron-Go utilityProcess check exit code = %d\nOutput:\n%s", electronGo.ExitCode, electronGo.Output)
	}

	electronReport := parseUtilityProcess(t, electron.Output)
	electronGoReport := parseUtilityProcess(t, electronGo.Output)
	if !reflect.DeepEqual(electronReport, electronGoReport) {
		t.Fatalf("utilityProcess report mismatch:\nelectron=%#v\nelectron-go=%#v", electronReport, electronGoReport)
	}
}

type fixtureResult struct {
	ExitCode int
	Output   string
}

type appActivityReport struct {
	Platform  string `json:"platform"`
	Supported bool   `json:"supported"`
	Active    bool   `json:"active"`
}

type clipboardReport struct {
	TextRoundTrip bool   `json:"textRoundTrip"`
	Text          string `json:"text"`
	Error         string `json:"error,omitempty"`
}

type clipboardFormatsReport struct {
	HTMLContainsPayload bool   `json:"htmlContainsPayload"`
	RTFRoundTrip        bool   `json:"rtfRoundTrip"`
	RTFBytes            int    `json:"rtfBytes"`
	BufferRoundTrip     bool   `json:"bufferRoundTrip"`
	BufferBytes         int    `json:"bufferBytes"`
	Error               string `json:"error,omitempty"`
}

type shellReport struct {
	MissingPathRejected  bool   `json:"missingPathRejected"`
	ErrorMessageNonempty bool   `json:"errorMessageNonempty"`
	Error                string `json:"error,omitempty"`
}

type contentTracingReport struct {
	Started               bool     `json:"started"`
	Stopped               bool     `json:"stopped"`
	RequestedPathReturned bool     `json:"requestedPathReturned"`
	Categories            []string `json:"categories"`
	HeapProfiling         bool     `json:"heapProfiling"`
	Error                 string   `json:"error,omitempty"`
}

type crashReporterReport struct {
	Started              bool   `json:"started"`
	UploadInitial        bool   `json:"uploadInitial"`
	UploadAfterSet       bool   `json:"uploadAfterSet"`
	ExtraInitial         bool   `json:"extraInitial"`
	ExtraAdded           bool   `json:"extraAdded"`
	ExtraRemoved         bool   `json:"extraRemoved"`
	UploadedReportsEmpty bool   `json:"uploadedReportsEmpty"`
	LastReportNull       bool   `json:"lastReportNull"`
	Error                string `json:"error,omitempty"`
}

type browserWindowOptionsReport struct {
	Width                    int    `json:"width"`
	Height                   int    `json:"height"`
	Visible                  bool   `json:"visible"`
	Title                    string `json:"title"`
	DevTools                 bool   `json:"devTools"`
	DefaultDevTools          bool   `json:"defaultDevTools"`
	DefaultContextIsolation  bool   `json:"defaultContextIsolation"`
	ExplicitContextIsolation bool   `json:"explicitContextIsolation"`
	ExplicitNodeIntegration  bool   `json:"explicitNodeIntegration"`
	ModalConstructed         bool   `json:"modalConstructed"`
	Error                    string `json:"error,omitempty"`
}

type browserWindowMethodsReport struct {
	InitialVisible            bool       `json:"initialVisible"`
	AfterShow                 bool       `json:"afterShow"`
	AfterHide                 bool       `json:"afterHide"`
	EventOrder                []string   `json:"eventOrder"`
	Bounds                    rectReport `json:"bounds"`
	NormalBounds              rectReport `json:"normalBounds"`
	EnabledAfterDisable       bool       `json:"enabledAfterDisable"`
	EnabledAfterEnable        bool       `json:"enabledAfterEnable"`
	MinimizedAfterMinimize    bool       `json:"minimizedAfterMinimize"`
	MinimizedAfterRestore     bool       `json:"minimizedAfterRestore"`
	MaximizedAfterMaximize    bool       `json:"maximizedAfterMaximize"`
	MaximizedAfterUnmaximize  bool       `json:"maximizedAfterUnmaximize"`
	ClosePrevented            bool       `json:"closePrevented"`
	UsableAfterPreventedClose bool       `json:"usableAfterPreventedClose"`
	CloseEvent                bool       `json:"closeEvent"`
	ClosedEvent               bool       `json:"closedEvent"`
	Closed                    bool       `json:"closed"`
	EnabledAfterEnd           bool       `json:"enabledAfterEnd"`
	Error                     string     `json:"error,omitempty"`
}

type viewTreeReport struct {
	BoundsChanged           bool       `json:"boundsChanged"`
	RootChildWidths         []int      `json:"rootChildWidths"`
	AfterRemoveWidths       []int      `json:"afterRemoveWidths"`
	AfterReparentWidths     []int      `json:"afterReparentWidths"`
	NewParentChildWidths    []int      `json:"newParentChildWidths"`
	LiveWebContentsLoaded   bool       `json:"liveWebContentsLoaded"`
	LiveWebContentsURL      string     `json:"liveWebContentsURL"`
	LiveWebContentsParented bool       `json:"liveWebContentsParented"`
	LiveWebContentsDetached bool       `json:"liveWebContentsDetached"`
	ProbeBounds             rectReport `json:"probeBounds"`
	Error                   string     `json:"error,omitempty"`
}

type viewAnimationReport struct {
	Animated                      bool       `json:"animated"`
	DurationMS                    int        `json:"durationMS"`
	Easing                        string     `json:"easing"`
	BoundsChanged                 bool       `json:"boundsChanged"`
	BackgroundBlurMethodAvailable bool       `json:"backgroundBlurMethodAvailable"`
	BackgroundBlurAccepted        bool       `json:"backgroundBlurAccepted"`
	Bounds                        rectReport `json:"bounds"`
	Error                         string     `json:"error,omitempty"`
}

type webContentsNavigationReport struct {
	DidStartNavigation bool   `json:"didStartNavigation"`
	DidNavigate        bool   `json:"didNavigate"`
	DidFinishLoad      bool   `json:"didFinishLoad"`
	DidNavigateInPage  bool   `json:"didNavigateInPage"`
	CanGoBack          bool   `json:"canGoBack"`
	CanGoForward       bool   `json:"canGoForward"`
	LoadingAfter       bool   `json:"loadingAfter"`
	FinalHash          string `json:"finalHash"`
	HistoryLength      int    `json:"historyLength"`
	Error              string `json:"error,omitempty"`
}

type webContentsDevToolsTargetReport struct {
	MethodAvailable bool   `json:"methodAvailable"`
	FirstNonempty   bool   `json:"firstNonempty"`
	Stable          bool   `json:"stable"`
	Distinct        bool   `json:"distinct"`
	Lookupable      bool   `json:"lookupable"`
	Error           string `json:"error,omitempty"`
}

type webContentsPrintReport struct {
	DefaultAccepted  bool   `json:"defaultAccepted"`
	DefaultPageSize  bool   `json:"defaultPageSize"`
	PageSizeOmitted  bool   `json:"pageSizeOmitted"`
	CopiesDefault    int    `json:"copiesDefault"`
	ConflictRejected bool   `json:"conflictRejected"`
	Error            string `json:"error,omitempty"`
}

type focusOnNavigationReport struct {
	DefaultAccepted       bool   `json:"defaultAccepted"`
	DefaultFocuses        bool   `json:"defaultFocuses"`
	ExplicitFalseAccepted bool   `json:"explicitFalseAccepted"`
	ExplicitFalseFocuses  bool   `json:"explicitFalseFocuses"`
	Error                 string `json:"error,omitempty"`
}

type offscreenDeviceScaleReport struct {
	DefaultAccepted          bool    `json:"defaultAccepted"`
	DefaultDeviceScaleFactor float64 `json:"defaultDeviceScaleFactor"`
	CustomAccepted           bool    `json:"customAccepted"`
	CustomDeviceScaleFactor  float64 `json:"customDeviceScaleFactor"`
	Error                    string  `json:"error,omitempty"`
}

type ipcReport struct {
	InvokePong             bool   `json:"invokePong"`
	ArgsEcho               bool   `json:"argsEcho"`
	OnceFirst              bool   `json:"onceFirst"`
	OnceSecondRejected     bool   `json:"onceSecondRejected"`
	RemovedRejected        bool   `json:"removedRejected"`
	DuplicateRejected      bool   `json:"duplicateRejected"`
	MissingRejected        bool   `json:"missingRejected"`
	MessagePortRoundTrip   bool   `json:"messagePortRoundTrip"`
	TransferredPortMessage bool   `json:"transferredPortMessage"`
	Error                  string `json:"error,omitempty"`
}

type contextBridgeReport struct {
	ExposedVersion    bool   `json:"exposedVersion"`
	FunctionCallable  bool   `json:"functionCallable"`
	ArrayValueCopied  bool   `json:"arrayValueCopied"`
	DuplicateRejected bool   `json:"duplicateRejected"`
	MutationRejected  bool   `json:"mutationRejected"`
	Error             string `json:"error,omitempty"`
}

type rectReport struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type appLifecycleReport struct {
	Ready               bool     `json:"ready"`
	WhenReady           bool     `json:"whenReady"`
	EventOrder          []string `json:"eventOrder"`
	CancelledBeforeQuit bool     `json:"cancelledBeforeQuit"`
	QuitExitCode        int      `json:"quitExitCode"`
	Error               string   `json:"error,omitempty"`
}

type appPathsReport struct {
	AppPathNonempty             bool   `json:"appPathNonempty"`
	HomePathNonempty            bool   `json:"homePathNonempty"`
	AppDataPathNonempty         bool   `json:"appDataPathNonempty"`
	UserDataPathNonempty        bool   `json:"userDataPathNonempty"`
	TempPathNonempty            bool   `json:"tempPathNonempty"`
	ExePathNonempty             bool   `json:"exePathNonempty"`
	ModulePathNonempty          bool   `json:"modulePathNonempty"`
	DesktopPathNonempty         bool   `json:"desktopPathNonempty"`
	DocumentsPathNonempty       bool   `json:"documentsPathNonempty"`
	DownloadsPathNonempty       bool   `json:"downloadsPathNonempty"`
	MusicPathNonempty           bool   `json:"musicPathNonempty"`
	PicturesPathNonempty        bool   `json:"picturesPathNonempty"`
	VideosPathNonempty          bool   `json:"videosPathNonempty"`
	CrashDumpsPathNonempty      bool   `json:"crashDumpsPathNonempty"`
	SessionDataDefaultsUserData bool   `json:"sessionDataDefaultsUserData"`
	SessionDataOverrideWorks    bool   `json:"sessionDataOverrideWorks"`
	LogsOverrideWorks           bool   `json:"logsOverrideWorks"`
	Error                       string `json:"error,omitempty"`
}

type asarReport struct {
	ReadMain           bool   `json:"readMain"`
	RequireIndex       bool   `json:"requireIndex"`
	UnpackedRead       bool   `json:"unpackedRead"`
	CopySourceUnpacked bool   `json:"copySourceUnpacked"`
	StatSize           int64  `json:"statSize"`
	Error              string `json:"error,omitempty"`
}

type appWebAuthnReport struct {
	Configured          bool   `json:"configured"`
	TouchIDSupported    bool   `json:"touchIDSupported"`
	KeychainGroupStored bool   `json:"keychainGroupStored"`
	InvalidRejected     bool   `json:"invalidRejected"`
	Error               string `json:"error,omitempty"`
}

type nodeTransformTypesReport struct {
	Supported                  bool   `json:"supported"`
	ExperimentalTransformTypes bool   `json:"experimentalTransformTypes"`
	ExecArgvIncludesFlag       bool   `json:"execArgvIncludesFlag"`
	Node                       string `json:"node"`
}

type nodeVersionsReport struct {
	Electron string `json:"electron"`
	Chrome   string `json:"chrome"`
	Node     string `json:"node"`
	V8       string `json:"v8"`
	Modules  string `json:"modules"`
}

type autoUpdaterReport struct {
	Platform        string `json:"platform"`
	Supported       bool   `json:"supported"`
	FeedURL         string `json:"feedURL"`
	DefaultProvider string `json:"defaultProvider,omitempty"`
	Error           string `json:"error,omitempty"`
}

type netLogReport struct {
	Started       bool   `json:"started"`
	ActiveDuring  bool   `json:"activeDuring"`
	Stopped       bool   `json:"stopped"`
	InactiveAfter bool   `json:"inactiveAfter"`
	Error         string `json:"error,omitempty"`
}

type clientRequestReport struct {
	Method         string `json:"method"`
	Path           string `json:"path"`
	QueryMode      string `json:"queryMode"`
	Header         string `json:"header"`
	Body           string `json:"body"`
	RedirectPolicy string `json:"redirectPolicy"`
	Ended          bool   `json:"ended"`
	Error          string `json:"error,omitempty"`
}

type globalShortcutSuspensionReport struct {
	InitialSuspended  bool   `json:"initialSuspended"`
	SuspendedAfterSet bool   `json:"suspendedAfterSet"`
	ResumedAfterUnset bool   `json:"resumedAfterUnset"`
	Error             string `json:"error,omitempty"`
}

type menuReport struct {
	ItemCount       int    `json:"itemCount"`
	FirstLabel      string `json:"firstLabel"`
	FirstEnabled    bool   `json:"firstEnabled"`
	CheckboxLabel   string `json:"checkboxLabel"`
	CheckboxChecked bool   `json:"checkboxChecked"`
	SubmenuFound    bool   `json:"submenuFound"`
	SubmenuRole     string `json:"submenuRole"`
	Error           string `json:"error,omitempty"`
}

type nativeThemeReport struct {
	Platform                          string `json:"platform"`
	ShouldDifferentiateWithoutColor   bool   `json:"shouldDifferentiateWithoutColor"`
	SupportsDifferentiateWithoutColor bool   `json:"supportsDifferentiateWithoutColor"`
}

type notificationReport struct {
	Title          string `json:"title"`
	Body           string `json:"body"`
	Silent         bool   `json:"silent"`
	DefaultUrgency string `json:"defaultUrgency"`
	Error          string `json:"error,omitempty"`
}

type notificationIdentityReport struct {
	Platform                   string `json:"platform"`
	Supported                  bool   `json:"supported"`
	HistoryAvailable           bool   `json:"historyAvailable"`
	RemoveFromHistoryAvailable bool   `json:"removeFromHistoryAvailable"`
	ID                         string `json:"id,omitempty"`
	GroupID                    string `json:"groupId,omitempty"`
	GroupTitle                 string `json:"groupTitle,omitempty"`
	Title                      string `json:"title,omitempty"`
	Body                       string `json:"body,omitempty"`
	Error                      string `json:"error,omitempty"`
}

type safeStorageReport struct {
	Available       bool   `json:"available"`
	AsyncAvailable  bool   `json:"asyncAvailable"`
	Backend         string `json:"backend"`
	RoundTrip       bool   `json:"roundTrip"`
	CiphertextBytes int    `json:"ciphertextBytes"`
	Error           string `json:"error,omitempty"`
}

type protocolReport struct {
	RegisteredPrivileged   bool   `json:"registeredPrivileged"`
	AllowExtensions        bool   `json:"allowExtensions"`
	HandledAfterRegister   bool   `json:"handledAfterRegister"`
	FetchStatus            int    `json:"fetchStatus"`
	FetchHeader            bool   `json:"fetchHeader"`
	FetchBody              bool   `json:"fetchBody"`
	DuplicateRejected      bool   `json:"duplicateRejected"`
	LatePrivilegedRejected bool   `json:"latePrivilegedRejected"`
	HandledAfterRemove     bool   `json:"handledAfterRemove"`
	Error                  string `json:"error,omitempty"`
}

type sessionReport struct {
	DefaultSameWithEmpty       bool   `json:"defaultSameWithEmpty"`
	PersistStoragePathNonempty bool   `json:"persistStoragePathNonempty"`
	MemoryStoragePathEmpty     bool   `json:"memoryStoragePathEmpty"`
	CookieRoundTrip            bool   `json:"cookieRoundTrip"`
	CookieCount                int    `json:"cookieCount"`
	CookieOverwriteValue       bool   `json:"cookieOverwriteValue"`
	CookieOverwriteChange      bool   `json:"cookieOverwriteChange"`
	CookieRemoveResolved       bool   `json:"cookieRemoveResolved"`
	CookieRemoved              bool   `json:"cookieRemoved"`
	CookieRemoveChange         bool   `json:"cookieRemoveChange"`
	PartitionCookieIsolation   bool   `json:"partitionCookieIsolation"`
	CacheClearResolved         bool   `json:"cacheClearResolved"`
	StorageClearResolved       bool   `json:"storageClearResolved"`
	StorageClearRemovedCookies bool   `json:"storageClearRemovedCookies"`
	Error                      string `json:"error,omitempty"`
}

type sessionQuotasReport struct {
	AcceptsQuotas bool   `json:"acceptsQuotas"`
	ErrorName     string `json:"errorName"`
	Error         string `json:"error,omitempty"`
}

type sessionWebAuthnReport struct {
	HandlerCalled      bool   `json:"handlerCalled"`
	RequestID          string `json:"requestId"`
	Origin             string `json:"origin"`
	CredentialCount    int    `json:"credentialCount"`
	SelectedCredential string `json:"selectedCredential"`
	Error              string `json:"error,omitempty"`
}

type utilityProcessReport struct {
	SpawnEvent       bool     `json:"spawnEvent"`
	MessageEvent     bool     `json:"messageEvent"`
	ExitEvent        bool     `json:"exitEvent"`
	ExitCode         int      `json:"exitCode"`
	StdioMode        string   `json:"stdioMode"`
	StdoutPiped      bool     `json:"stdoutPiped"`
	StderrPiped      bool     `json:"stderrPiped"`
	StdinPiped       bool     `json:"stdinPiped"`
	StdinWrite       bool     `json:"stdinWrite"`
	Args             []string `json:"args"`
	EnvironmentValue string   `json:"environmentValue"`
	ServiceName      string   `json:"serviceName"`
	Error            string   `json:"error,omitempty"`
}

func runFixture(t *testing.T, command, fixture string) fixtureResult {
	t.Helper()
	skipSandboxedDarwinElectron(t, command)
	return runCommand(t, command, fixture)
}

func runFixtureWithEnv(t *testing.T, command, fixture string, env ...string) fixtureResult {
	t.Helper()
	skipSandboxedDarwinElectron(t, command)
	return runCommandWithEnv(t, command, env, fixture)
}

func skipSandboxedDarwinElectron(t *testing.T, command string) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		return
	}
	if os.Getenv("CODEX_SANDBOX") == "" {
		return
	}
	if os.Getenv("ELECTRON_GO_ALLOW_SANDBOXED_DARWIN_CONFORMANCE") == "1" {
		return
	}
	if !strings.Contains(strings.ToLower(command), "electron") {
		return
	}
	t.Skip("official Electron GUI launch aborts under the macOS seatbelt sandbox; run outside the sandbox or set ELECTRON_GO_ALLOW_SANDBOXED_DARWIN_CONFORMANCE=1 to force it")
}

func runCommand(t *testing.T, command string, args ...string) fixtureResult {
	t.Helper()
	return runCommandWithEnv(t, command, nil, args...)
}

func runCommandWithEnv(t *testing.T, command string, env []string, args ...string) fixtureResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	parts := strings.Fields(command)
	if len(parts) == 0 {
		t.Fatalf("empty command")
	}

	cmd := exec.CommandContext(ctx, parts[0], append(parts[1:], args...)...)
	cmd.Env = os.Environ()
	if len(env) > 0 {
		cmd.Env = append(cmd.Env, env...)
	}
	prepareCommandForCleanup(cmd)
	cmd.Cancel = func() error {
		terminateCommandGroup(cmd)
		return nil
	}
	cmd.WaitDelay = 2 * time.Second
	output, err := cmd.CombinedOutput()
	result := fixtureResult{Output: string(output)}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	} else {
		result.ExitCode = -1
	}
	if err != nil && result.ExitCode == 0 {
		t.Fatalf("%s failed without process state: %v", command, err)
	}
	return result
}

func parseAppActivity(t *testing.T, output string) appActivityReport {
	t.Helper()
	var report appActivityReport
	if err := json.Unmarshal([]byte(lastJSONLine(output)), &report); err != nil {
		t.Fatalf("app activity output is not JSON: %v\n%s", err, output)
	}
	if report.Platform == "" {
		t.Fatalf("app activity report platform is empty: %#v", report)
	}
	return report
}

func parseClipboard(t *testing.T, output string) clipboardReport {
	t.Helper()
	var report clipboardReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("clipboard output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("clipboard report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseClipboardFormats(t *testing.T, output string) clipboardFormatsReport {
	t.Helper()
	var report clipboardFormatsReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("clipboard formats output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("clipboard formats report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseShell(t *testing.T, output string) shellReport {
	t.Helper()
	var report shellReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("shell output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("shell report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseContentTracing(t *testing.T, output string) contentTracingReport {
	t.Helper()
	var report contentTracingReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("contentTracing output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("contentTracing report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseCrashReporter(t *testing.T, output string) crashReporterReport {
	t.Helper()
	var report crashReporterReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("crashReporter output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("crashReporter report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseBrowserWindowOptions(t *testing.T, output string) browserWindowOptionsReport {
	t.Helper()
	var report browserWindowOptionsReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("BrowserWindow options output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("BrowserWindow options report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseBrowserWindowMethods(t *testing.T, output string) browserWindowMethodsReport {
	t.Helper()
	var report browserWindowMethodsReport
	if err := json.Unmarshal([]byte(lastJSONLine(output)), &report); err != nil {
		t.Fatalf("BrowserWindow methods output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("BrowserWindow methods report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseViewTree(t *testing.T, output string) viewTreeReport {
	t.Helper()
	var report viewTreeReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("View tree output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("View tree report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseViewAnimation(t *testing.T, output string) viewAnimationReport {
	t.Helper()
	var report viewAnimationReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("View animation output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("View animation report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseWebContentsNavigation(t *testing.T, output string) webContentsNavigationReport {
	t.Helper()
	var report webContentsNavigationReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("webContents navigation output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("webContents navigation report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseWebContentsDevToolsTarget(t *testing.T, output string) webContentsDevToolsTargetReport {
	t.Helper()
	var report webContentsDevToolsTargetReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("webContents DevTools target output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("webContents DevTools target report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseWebContentsPrint(t *testing.T, output string) webContentsPrintReport {
	t.Helper()
	var report webContentsPrintReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("webContents print output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("webContents print report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseFocusOnNavigation(t *testing.T, output string) focusOnNavigationReport {
	t.Helper()
	var report focusOnNavigationReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("focusOnNavigation output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("focusOnNavigation report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseOffscreenDeviceScale(t *testing.T, output string) offscreenDeviceScaleReport {
	t.Helper()
	var report offscreenDeviceScaleReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("offscreen device scale output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("offscreen device scale report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseIPC(t *testing.T, output string) ipcReport {
	t.Helper()
	var report ipcReport
	if err := json.Unmarshal([]byte(lastJSONLine(output)), &report); err != nil {
		t.Fatalf("IPC output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("IPC report has error %q\n%s", report.Error, output)
	}
	return report
}

func lastJSONLine(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			return line
		}
	}
	return strings.TrimSpace(output)
}

func parseContextBridge(t *testing.T, output string) contextBridgeReport {
	t.Helper()
	var report contextBridgeReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("contextBridge output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("contextBridge report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseAppLifecycle(t *testing.T, output string) appLifecycleReport {
	t.Helper()
	var report appLifecycleReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("app lifecycle output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("app lifecycle report has error %q\n%s", report.Error, output)
	}
	if len(report.EventOrder) == 0 {
		t.Fatalf("app lifecycle eventOrder is empty: %#v", report)
	}
	return report
}

func parseAppPaths(t *testing.T, output string) appPathsReport {
	t.Helper()
	var report appPathsReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("app paths output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("app paths report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseAsar(t *testing.T, output string) asarReport {
	t.Helper()
	var report asarReport
	if err := json.Unmarshal([]byte(lastJSONLine(output)), &report); err != nil {
		t.Fatalf("ASAR output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("ASAR report has error %q\n%s", report.Error, output)
	}
	return report
}

const asarFixtureArchiveBase64 = "BAAAABAEAAAMBAAABQQAAHsiZmlsZXMiOnsibWFpbi5qcyI6eyJzaXplIjoxOSwib2Zmc2V0IjoiMCIsImludGVncml0eSI6eyJhbGdvcml0aG0iOiJTSEEyNTYiLCJoYXNoIjoiZjg3N2E4NjlkODI3YTdiOTg5Yjc4MzMwNTM4ZDEwNTRlOTk4MGM1MTlhMDE3NjU4ZjIzMzg3NjY1MzU1Zjg1MSIsImJsb2NrU2l6ZSI6NDE5NDMwNCwiYmxvY2tzIjpbImY4NzdhODY5ZDgyN2E3Yjk4OWI3ODMzMDUzOGQxMDU0ZTk5ODBjNTE5YTAxNzY1OGYyMzM4NzY2NTM1NWY4NTEiXX19LCJuYXRpdmUiOnsiZmlsZXMiOnsiYWRkb24ubm9kZSI6eyJzaXplIjo2LCJvZmZzZXQiOiIxOSIsImludGVncml0eSI6eyJhbGdvcml0aG0iOiJTSEEyNTYiLCJoYXNoIjoiYmVmMzJkMmMzMTVhMjg5NTc2ZjJhNjgyOGQyN2VkYjE2YmIzMTZhNGQ4NWMyNzFmMmQ3OTQwNDVmM2VhNjY4ZCIsImJsb2NrU2l6ZSI6NDE5NDMwNCwiYmxvY2tzIjpbImJlZjMyZDJjMzE1YTI4OTU3NmYyYTY4MjhkMjdlZGIxNmJiMzE2YTRkODVjMjcxZjJkNzk0MDQ1ZjNlYTY2OGQiXX19fX0sInBhY2thZ2UuanNvbiI6eyJzaXplIjozNSwib2Zmc2V0IjoiMjUiLCJpbnRlZ3JpdHkiOnsiYWxnb3JpdGhtIjoiU0hBMjU2IiwiaGFzaCI6IjExZmRlMWNjYzczYzdmMTI3OTlhYmM2OTNhYTBhYjM2NjRlYjFiYmE3ZjgwZjgzODllMzdhNDNkZmYxMGNjNzIiLCJibG9ja1NpemUiOjQxOTQzMDQsImJsb2NrcyI6WyIxMWZkZTFjY2M3M2M3ZjEyNzk5YWJjNjkzYWEwYWIzNjY0ZWIxYmJhN2Y4MGY4Mzg5ZTM3YTQzZGZmMTBjYzcyIl19fSwicGtnIjp7ImZpbGVzIjp7ImluZGV4LmpzIjp7InNpemUiOjE5LCJvZmZzZXQiOiI2MCIsImludGVncml0eSI6eyJhbGdvcml0aG0iOiJTSEEyNTYiLCJoYXNoIjoiNDdlZjRmNjYzM2M3NjEwMzgxNDE3MzIwNDM0ZTliZDdhNjVhYmYwODMyNjk0ZmJjN2Q4ODc5OTFiNDc4NzlhZSIsImJsb2NrU2l6ZSI6NDE5NDMwNCwiYmxvY2tzIjpbIjQ3ZWY0ZjY2MzNjNzYxMDM4MTQxNzMyMDQzNGU5YmQ3YTY1YWJmMDgzMjY5NGZiYzdkODg3OTkxYjQ3ODc5YWUiXX19fX19fQAAAGNvbnNvbGUubG9nKCdhc2FyJyluYXRpdmV7Im5hbWUiOiJmaXh0dXJlIiwibWFpbiI6Im1haW4uanMifW1vZHVsZS5leHBvcnRzID0gNDI="

func writeAsarFixtureArchive(t *testing.T) string {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(asarFixtureArchiveBase64)
	if err != nil {
		t.Fatalf("decode ASAR fixture archive: %v", err)
	}
	path := filepath.Join(t.TempDir(), "app.asar")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write ASAR fixture archive: %v", err)
	}
	return path
}

func parseAppWebAuthn(t *testing.T, output string) appWebAuthnReport {
	t.Helper()
	var report appWebAuthnReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("app WebAuthn output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("app WebAuthn report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseAutoUpdater(t *testing.T, output string) autoUpdaterReport {
	t.Helper()
	var report autoUpdaterReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("autoUpdater output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("autoUpdater report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseNodeTransformTypes(t *testing.T, output string) nodeTransformTypesReport {
	t.Helper()
	var report nodeTransformTypesReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("node transform-types output is not JSON: %v\n%s", err, output)
	}
	if report.Node == "" {
		t.Fatalf("node transform-types report node is empty: %#v", report)
	}
	return report
}

func parseNodeVersions(t *testing.T, output string) nodeVersionsReport {
	t.Helper()
	var report nodeVersionsReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("node versions output is not JSON: %v\n%s", err, output)
	}
	if report.Electron == "" || report.Chrome == "" || report.Node == "" || report.V8 == "" || report.Modules == "" {
		t.Fatalf("node versions report has empty fields: %#v", report)
	}
	return report
}

func parseNetLog(t *testing.T, output string) netLogReport {
	t.Helper()
	var report netLogReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("netLog output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("netLog report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseClientRequest(t *testing.T, output string) clientRequestReport {
	t.Helper()
	var report clientRequestReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("ClientRequest output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("ClientRequest report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseGlobalShortcutSuspension(t *testing.T, output string) globalShortcutSuspensionReport {
	t.Helper()
	var report globalShortcutSuspensionReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("globalShortcut suspension output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("globalShortcut suspension report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseMenu(t *testing.T, output string) menuReport {
	t.Helper()
	var report menuReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("menu output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("menu report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseNativeTheme(t *testing.T, output string) nativeThemeReport {
	t.Helper()
	var report nativeThemeReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("nativeTheme output is not JSON: %v\n%s", err, output)
	}
	if report.Platform == "" {
		t.Fatalf("nativeTheme platform is empty: %#v", report)
	}
	if !report.SupportsDifferentiateWithoutColor && report.ShouldDifferentiateWithoutColor {
		t.Fatalf("unsupported nativeTheme reported shouldDifferentiateWithoutColor=true: %#v", report)
	}
	return report
}

func parseNotification(t *testing.T, output string) notificationReport {
	t.Helper()
	var report notificationReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("notification output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("notification report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseNotificationIdentity(t *testing.T, output string) notificationIdentityReport {
	t.Helper()
	var report notificationIdentityReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("notification identity output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("notification identity report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseSafeStorage(t *testing.T, output string) safeStorageReport {
	t.Helper()
	var report safeStorageReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("safeStorage output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("safeStorage report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseProtocol(t *testing.T, output string) protocolReport {
	t.Helper()
	var report protocolReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("protocol output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("protocol report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseSession(t *testing.T, output string) sessionReport {
	t.Helper()
	var report sessionReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("session output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("session report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseSessionQuotas(t *testing.T, output string) sessionQuotasReport {
	t.Helper()
	var report sessionQuotasReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("session quotas output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("session quotas report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseSessionWebAuthn(t *testing.T, output string) sessionWebAuthnReport {
	t.Helper()
	var report sessionWebAuthnReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("session WebAuthn output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("session WebAuthn report has error %q\n%s", report.Error, output)
	}
	return report
}

func parseUtilityProcess(t *testing.T, output string) utilityProcessReport {
	t.Helper()
	var report utilityProcessReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &report); err != nil {
		t.Fatalf("utilityProcess output is not JSON: %v\n%s", err, output)
	}
	if report.Error != "" {
		t.Fatalf("utilityProcess report has error %q\n%s", report.Error, output)
	}
	return report
}

func equalIntSlices(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
