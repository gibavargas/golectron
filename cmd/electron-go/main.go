package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	egapplifecycle "github.com/gibavargas/electron-go/internal/applifecycle"
	egapppaths "github.com/gibavargas/electron-go/internal/apppaths"
	egasar "github.com/gibavargas/electron-go/internal/asar"
	egautoupdater "github.com/gibavargas/electron-go/internal/autoupdater"
	egbrowserwindow "github.com/gibavargas/electron-go/internal/browserwindow"
	"github.com/gibavargas/electron-go/internal/compat"
	egcontextbridge "github.com/gibavargas/electron-go/internal/contextbridge"
	egdesktop "github.com/gibavargas/electron-go/internal/desktop"
	egdiagnostics "github.com/gibavargas/electron-go/internal/diagnostics"
	egglobalshortcut "github.com/gibavargas/electron-go/internal/globalshortcut"
	egipc "github.com/gibavargas/electron-go/internal/ipc"
	egmenu "github.com/gibavargas/electron-go/internal/menu"
	"github.com/gibavargas/electron-go/internal/native"
	egnativetheme "github.com/gibavargas/electron-go/internal/nativetheme"
	egnet "github.com/gibavargas/electron-go/internal/netrequest"
	egnodecompat "github.com/gibavargas/electron-go/internal/nodecompat"
	egnotification "github.com/gibavargas/electron-go/internal/notification"
	egprotocol "github.com/gibavargas/electron-go/internal/protocol"
	egruntime "github.com/gibavargas/electron-go/internal/runtime"
	egstorage "github.com/gibavargas/electron-go/internal/safestorage"
	egsession "github.com/gibavargas/electron-go/internal/session"
	egutilityprocess "github.com/gibavargas/electron-go/internal/utilityprocess"
	egview "github.com/gibavargas/electron-go/internal/view"
	egwebauthn "github.com/gibavargas/electron-go/internal/webauthn"
	egwebcontents "github.com/gibavargas/electron-go/internal/webcontents"
)

const version = "0.1.0"
const helloFixtureDir = "compat/fixtures/hello"

func main() {
	goruntime.LockOSThread()
	code := run(os.Args, os.Environ())
	os.Exit(code)
}

func run(argv []string, env []string) int {
	ctx := context.Background()
	subprocess, err := egruntime.ExecuteSubprocess(ctx, egruntime.NativeSubprocessHook, argv, env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
		return 1
	}
	if subprocess.Handled {
		return subprocess.ExitCode
	}

	args := []string(nil)
	if len(argv) > 1 {
		args = argv[1:]
	}

	fs := flag.NewFlagSet("electron-go", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	showVersion := fs.Bool("version", false, "print the Electron-Go version")
	showLedger := fs.Bool("compat-json", false, "print the compatibility ledger as JSON")
	showE2EAudit := fs.Bool("e2e-audit", false, "print compatible ledger items lacking e2e/conformance evidence")
	checkParity := fs.Bool("check-parity", false, "exit successfully only when every ledger item is compatible")
	browserWindowOptionsCheck := fs.Bool("browser-window-options-check", false, "print BrowserWindow constructor option compatibility JSON")
	browserWindowMethodsCheck := fs.Bool("browser-window-methods-check", false, "print BrowserWindow methods compatibility JSON")
	viewTreeCheck := fs.Bool("view-tree-check", false, "print BaseWindow/View tree compatibility JSON")
	viewAnimationCheck := fs.Bool("view-animation-check", false, "print View animated bounds compatibility JSON")
	webContentsNavigationCheck := fs.Bool("webcontents-navigation-check", false, "print webContents navigation compatibility JSON")
	webContentsDevToolsTargetCheck := fs.Bool("webcontents-devtools-target-check", false, "print webContents DevTools target id compatibility JSON")
	webContentsPrintCheck := fs.Bool("webcontents-print-check", false, "print webContents print option compatibility JSON")
	focusOnNavigationCheck := fs.Bool("focus-on-navigation-check", false, "print webPreferences.focusOnNavigation compatibility JSON")
	offscreenDeviceScaleCheck := fs.Bool("offscreen-device-scale-check", false, "print webPreferences.offscreen.deviceScaleFactor compatibility JSON")
	ipcCheck := fs.Bool("ipc-check", false, "print ipcMain/ipcRenderer invoke compatibility JSON")
	contextBridgeCheck := fs.Bool("context-bridge-check", false, "print contextBridge/preload compatibility JSON")
	cefInitCheck := fs.Bool("cef-init-check", false, "initialize and shut down CEF without opening a window")
	clipboardCheck := fs.Bool("clipboard-check", false, "print clipboard compatibility JSON")
	clipboardFormatsCheck := fs.Bool("clipboard-formats-check", false, "print clipboard rich-format compatibility JSON")
	shellCheck := fs.Bool("shell-check", false, "print shell compatibility JSON")
	contentTracingCheck := fs.Bool("content-tracing-check", false, "print contentTracing compatibility JSON")
	processModelCheck := fs.Bool("process-model-check", false, "verify CEF process ownership and subprocess teardown")
	appLifecycleCheck := fs.Bool("app-lifecycle-check", false, "print app lifecycle compatibility JSON")
	appPathsCheck := fs.Bool("app-paths-check", false, "print app path compatibility JSON")
	asarCheck := fs.Bool("asar-check", false, "print ASAR archive compatibility JSON")
	appWebAuthnCheck := fs.Bool("app-webauthn-check", false, "print app.configureWebAuthn compatibility JSON")
	appIsActiveCheck := fs.Bool("app-is-active-check", false, "print app.isActive() foreground-state compatibility JSON")
	autoUpdaterCheck := fs.Bool("auto-updater-check", false, "print autoUpdater compatibility JSON")
	globalShortcutSuspensionCheck := fs.Bool("global-shortcut-suspension-check", false, "print globalShortcut suspension compatibility JSON")
	menuCheck := fs.Bool("menu-check", false, "print menu template compatibility JSON")
	nativeThemeCheck := fs.Bool("native-theme-check", false, "print nativeTheme compatibility JSON")
	netLogCheck := fs.Bool("netlog-check", false, "print netLog lifecycle compatibility JSON")
	clientRequestCheck := fs.Bool("client-request-check", false, "print ClientRequest option compatibility JSON")
	nodeOptionsCheck := fs.Bool("node-options-check", false, "print embedded Node option compatibility JSON")
	nodeVersionsCheck := fs.Bool("node-versions-check", false, "print process.versions compatibility JSON")
	notificationCheck := fs.Bool("notification-check", false, "print notification compatibility JSON")
	notificationIdentityCheck := fs.Bool("notification-identity-check", false, "print Notification id/group compatibility JSON")
	protocolCheck := fs.Bool("protocol-check", false, "print protocol registration compatibility JSON")
	safeStorageCheck := fs.Bool("safe-storage-check", false, "print safeStorage compatibility JSON")
	sessionCheck := fs.Bool("session-check", false, "print session compatibility JSON")
	sessionQuotasCheck := fs.Bool("session-quotas-check", false, "print session.clearStorageData quotas compatibility JSON")
	sessionWebAuthnCheck := fs.Bool("session-webauthn-check", false, "print session select-webauthn-account compatibility JSON")
	utilityProcessCheck := fs.Bool("utility-process-check", false, "print utilityProcess compatibility JSON")
	runHello := fs.Bool("hello", false, "run the bundled hello conformance fixture")
	experimentalTransformTypes := fs.Bool("experimental-transform-types", false, "pass --experimental-transform-types to the embedded Node runtime")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintf(os.Stdout, "electron-go %s\n", version)
		fmt.Fprintf(os.Stdout, "electron baseline %s\n", compat.Target().Electron)
		return 0
	}

	ledger := compat.MustLoadLedger()
	if *showLedger {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(ledger); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		return 0
	}

	if *showE2EAudit {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(ledger.AuditE2EEvidence()); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		return 0
	}

	if *checkParity {
		audit := ledger.AuditE2EEvidence()
		if ledger.IsComplete() && audit.Pass {
			fmt.Fprintf(os.Stdout, "electron-go parity complete for Electron %s\n", ledger.Target.Electron)
			return 0
		}
		if ledger.IsComplete() && !audit.Pass {
			fmt.Fprintf(os.Stderr, "electron-go parity lacks e2e evidence for Electron %s: %d compatible items missing e2e proof\n", ledger.Target.Electron, len(audit.Missing))
			return 1
		}
		fmt.Fprintf(os.Stderr, "electron-go parity incomplete for Electron %s: %d/%d compatible\n", ledger.Target.Electron, ledger.CompatibleCount(), ledger.TotalCount())
		return 1
	}

	if *browserWindowOptionsCheck {
		report := browserWindowOptionsReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *browserWindowMethodsCheck {
		report := browserWindowMethodsReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *viewTreeCheck {
		report := viewTreeReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *viewAnimationCheck {
		report := viewAnimationReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *webContentsNavigationCheck {
		report := webContentsNavigationReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *webContentsDevToolsTargetCheck {
		report := webContentsDevToolsTargetReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *webContentsPrintCheck {
		report := webContentsPrintReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *focusOnNavigationCheck {
		report := focusOnNavigationReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *offscreenDeviceScaleCheck {
		report := offscreenDeviceScaleReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *ipcCheck {
		report := ipcReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *contextBridgeCheck {
		report := contextBridgeReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *cefInitCheck {
		appDir := "."
		if fs.NArg() > 0 {
			appDir = fs.Arg(0)
		}
		if err := native.CheckCEFInitialize(ctx, appDir, argv); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: cef init check failed: %v\n", err)
			return 1
		}
		fmt.Fprintln(os.Stdout, "electron-go: cef_initialize returned 1")
		return 0
	}

	if *clipboardCheck {
		report := clipboardReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *clipboardFormatsCheck {
		report := clipboardFormatsReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *shellCheck {
		report := shellReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *contentTracingCheck {
		report := contentTracingReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *processModelCheck {
		appDir := "."
		if fs.NArg() > 0 {
			appDir = fs.Arg(0)
		}
		report, err := native.CheckRuntimeProcessModel(ctx, appDir, argv)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if encodeErr := enc.Encode(report); encodeErr != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", encodeErr)
			return 1
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: process model check failed: %v\n", err)
			return 1
		}
		return 0
	}

	if *appLifecycleCheck {
		report := appLifecycleReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *appPathsCheck {
		report := appPathsReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *asarCheck {
		report := asarReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *appWebAuthnCheck {
		report := appWebAuthnReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *appIsActiveCheck {
		report, err := native.IsAppActive(ctx)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if encodeErr := enc.Encode(report); encodeErr != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", encodeErr)
			return 1
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: app.isActive check failed: %v\n", err)
			return 1
		}
		return 0
	}

	if *autoUpdaterCheck {
		report := autoUpdaterReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *globalShortcutSuspensionCheck {
		report := globalShortcutSuspensionReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *menuCheck {
		report := menuReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *nativeThemeCheck {
		report, err := egnativetheme.Snapshot(ctx)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if encodeErr := enc.Encode(report); encodeErr != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", encodeErr)
			return 1
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: nativeTheme check failed: %v\n", err)
			return 1
		}
		return 0
	}

	if *netLogCheck {
		report := netLogReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *clientRequestCheck {
		report := clientRequestReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	nodeOptions := egruntime.NodeOptions{
		ExperimentalTransformTypes: *experimentalTransformTypes || egruntime.ExtractNodeOptions(argv).ExperimentalTransformTypes,
	}
	if *nodeOptionsCheck {
		report := struct {
			Supported                  bool   `json:"supported"`
			ExperimentalTransformTypes bool   `json:"experimentalTransformTypes"`
			ExecArgvIncludesFlag       bool   `json:"execArgvIncludesFlag"`
			Node                       string `json:"node"`
		}{
			Supported:                  true,
			ExperimentalTransformTypes: nodeOptions.ExperimentalTransformTypes,
			ExecArgvIncludesFlag:       false,
			Node:                       ledger.Target.Node,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		return 0
	}

	if *nodeVersionsCheck {
		report, err := nodeVersionsReport(ledger.Target)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if encodeErr := enc.Encode(report); encodeErr != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", encodeErr)
			return 1
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: node versions check failed: %v\n", err)
			return 1
		}
		return 0
	}

	if *notificationCheck {
		report := notificationReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *notificationIdentityCheck {
		report := notificationIdentityReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *safeStorageCheck {
		report := safeStorageReport(ctx)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if !report.RoundTrip {
			return 1
		}
		return 0
	}

	if *protocolCheck {
		report := protocolReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *sessionCheck {
		report := sessionReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *sessionQuotasCheck {
		report := sessionQuotasReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *sessionWebAuthnCheck {
		report := sessionWebAuthnReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	if *utilityProcessCheck {
		report := utilityProcessReport()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 1
		}
		if report.Error != "" {
			return 1
		}
		return 0
	}

	appDir := "."
	if *runHello {
		appDir = findHelloFixtureDir()
	} else if fs.NArg() > 0 {
		appDir = fs.Arg(0)
	}

	rt := egruntime.New(egruntime.Options{
		AppDir:          appDir,
		ElectronVersion: ledger.Target.Electron,
		Bridge:          native.NewBridge(),
		Args:            argv,
		Environment:     env,
		NodeOptions:     nodeOptions,
		Out:             os.Stdout,
	})

	if err := rt.Run(ctx); err != nil {
		var subprocessExit *egruntime.SubprocessExit
		if errors.As(err, &subprocessExit) {
			return subprocessExit.Code
		}
		if errors.Is(err, native.ErrBridgeUnavailable) {
			fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
			return 78
		}
		fmt.Fprintf(os.Stderr, "electron-go: %v\n", err)
		return 1
	}

	return 0
}

type safeStorageCheckReport struct {
	Available       bool   `json:"available"`
	AsyncAvailable  bool   `json:"asyncAvailable"`
	Backend         string `json:"backend"`
	RoundTrip       bool   `json:"roundTrip"`
	CiphertextBytes int    `json:"ciphertextBytes"`
	Error           string `json:"error,omitempty"`
}

type netLogCheckReport struct {
	Started       bool   `json:"started"`
	ActiveDuring  bool   `json:"activeDuring"`
	Stopped       bool   `json:"stopped"`
	InactiveAfter bool   `json:"inactiveAfter"`
	Error         string `json:"error,omitempty"`
}

type clientRequestCheckReport struct {
	Method         string `json:"method"`
	Path           string `json:"path"`
	QueryMode      string `json:"queryMode"`
	Header         string `json:"header"`
	Body           string `json:"body"`
	RedirectPolicy string `json:"redirectPolicy"`
	Ended          bool   `json:"ended"`
	Error          string `json:"error,omitempty"`
}

type clipboardCheckReport struct {
	TextRoundTrip bool   `json:"textRoundTrip"`
	Text          string `json:"text"`
	Error         string `json:"error,omitempty"`
}

type clipboardFormatsCheckReport struct {
	HTMLContainsPayload bool   `json:"htmlContainsPayload"`
	RTFRoundTrip        bool   `json:"rtfRoundTrip"`
	RTFBytes            int    `json:"rtfBytes"`
	BufferRoundTrip     bool   `json:"bufferRoundTrip"`
	BufferBytes         int    `json:"bufferBytes"`
	Error               string `json:"error,omitempty"`
}

type shellCheckReport struct {
	MissingPathRejected  bool   `json:"missingPathRejected"`
	ErrorMessageNonempty bool   `json:"errorMessageNonempty"`
	Error                string `json:"error,omitempty"`
}

type contentTracingCheckReport struct {
	Started               bool     `json:"started"`
	Stopped               bool     `json:"stopped"`
	RequestedPathReturned bool     `json:"requestedPathReturned"`
	Categories            []string `json:"categories"`
	HeapProfiling         bool     `json:"heapProfiling"`
	Error                 string   `json:"error,omitempty"`
}

type notificationCheckReport struct {
	Title          string `json:"title"`
	Body           string `json:"body"`
	Silent         bool   `json:"silent"`
	DefaultUrgency string `json:"defaultUrgency"`
	Error          string `json:"error,omitempty"`
}

type notificationIdentityCheckReport struct {
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

type nodeVersionsCheckReport struct {
	Electron string `json:"electron"`
	Chrome   string `json:"chrome"`
	Node     string `json:"node"`
	V8       string `json:"v8"`
	Modules  string `json:"modules"`
}

type browserWindowOptionsCheckReport struct {
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

type browserWindowMethodsCheckReport struct {
	InitialVisible            bool     `json:"initialVisible"`
	AfterShow                 bool     `json:"afterShow"`
	AfterHide                 bool     `json:"afterHide"`
	EventOrder                []string `json:"eventOrder"`
	Bounds                    rectJSON `json:"bounds"`
	NormalBounds              rectJSON `json:"normalBounds"`
	EnabledAfterDisable       bool     `json:"enabledAfterDisable"`
	EnabledAfterEnable        bool     `json:"enabledAfterEnable"`
	MinimizedAfterMinimize    bool     `json:"minimizedAfterMinimize"`
	MinimizedAfterRestore     bool     `json:"minimizedAfterRestore"`
	MaximizedAfterMaximize    bool     `json:"maximizedAfterMaximize"`
	MaximizedAfterUnmaximize  bool     `json:"maximizedAfterUnmaximize"`
	ClosePrevented            bool     `json:"closePrevented"`
	UsableAfterPreventedClose bool     `json:"usableAfterPreventedClose"`
	CloseEvent                bool     `json:"closeEvent"`
	ClosedEvent               bool     `json:"closedEvent"`
	Closed                    bool     `json:"closed"`
	EnabledAfterEnd           bool     `json:"enabledAfterEnd"`
	Error                     string   `json:"error,omitempty"`
}

type viewTreeCheckReport struct {
	BoundsChanged           bool     `json:"boundsChanged"`
	RootChildWidths         []int    `json:"rootChildWidths"`
	AfterRemoveWidths       []int    `json:"afterRemoveWidths"`
	AfterReparentWidths     []int    `json:"afterReparentWidths"`
	NewParentChildWidths    []int    `json:"newParentChildWidths"`
	LiveWebContentsLoaded   bool     `json:"liveWebContentsLoaded"`
	LiveWebContentsURL      string   `json:"liveWebContentsURL"`
	LiveWebContentsParented bool     `json:"liveWebContentsParented"`
	LiveWebContentsDetached bool     `json:"liveWebContentsDetached"`
	ProbeBounds             rectJSON `json:"probeBounds"`
	Error                   string   `json:"error,omitempty"`
}

type viewAnimationCheckReport struct {
	Animated                      bool     `json:"animated"`
	DurationMS                    int      `json:"durationMS"`
	Easing                        string   `json:"easing"`
	BoundsChanged                 bool     `json:"boundsChanged"`
	BackgroundBlurMethodAvailable bool     `json:"backgroundBlurMethodAvailable"`
	BackgroundBlurAccepted        bool     `json:"backgroundBlurAccepted"`
	Bounds                        rectJSON `json:"bounds"`
	Error                         string   `json:"error,omitempty"`
}

type webContentsNavigationCheckReport struct {
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

type webContentsDevToolsTargetCheckReport struct {
	MethodAvailable bool   `json:"methodAvailable"`
	FirstNonempty   bool   `json:"firstNonempty"`
	Stable          bool   `json:"stable"`
	Distinct        bool   `json:"distinct"`
	Lookupable      bool   `json:"lookupable"`
	Error           string `json:"error,omitempty"`
}

type webContentsPrintCheckReport struct {
	DefaultAccepted  bool   `json:"defaultAccepted"`
	DefaultPageSize  bool   `json:"defaultPageSize"`
	PageSizeOmitted  bool   `json:"pageSizeOmitted"`
	CopiesDefault    int    `json:"copiesDefault"`
	ConflictRejected bool   `json:"conflictRejected"`
	Error            string `json:"error,omitempty"`
}

type focusOnNavigationCheckReport struct {
	DefaultAccepted       bool   `json:"defaultAccepted"`
	DefaultFocuses        bool   `json:"defaultFocuses"`
	ExplicitFalseAccepted bool   `json:"explicitFalseAccepted"`
	ExplicitFalseFocuses  bool   `json:"explicitFalseFocuses"`
	Error                 string `json:"error,omitempty"`
}

type offscreenDeviceScaleCheckReport struct {
	DefaultAccepted          bool    `json:"defaultAccepted"`
	DefaultDeviceScaleFactor float64 `json:"defaultDeviceScaleFactor"`
	CustomAccepted           bool    `json:"customAccepted"`
	CustomDeviceScaleFactor  float64 `json:"customDeviceScaleFactor"`
	Error                    string  `json:"error,omitempty"`
}

type ipcCheckReport struct {
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

type contextBridgeCheckReport struct {
	ExposedVersion    bool   `json:"exposedVersion"`
	FunctionCallable  bool   `json:"functionCallable"`
	ArrayValueCopied  bool   `json:"arrayValueCopied"`
	DuplicateRejected bool   `json:"duplicateRejected"`
	MutationRejected  bool   `json:"mutationRejected"`
	Error             string `json:"error,omitempty"`
}

type rectJSON struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type appPathsCheckReport struct {
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

type asarCheckReport struct {
	ReadMain           bool   `json:"readMain"`
	RequireIndex       bool   `json:"requireIndex"`
	UnpackedRead       bool   `json:"unpackedRead"`
	CopySourceUnpacked bool   `json:"copySourceUnpacked"`
	StatSize           int64  `json:"statSize"`
	Error              string `json:"error,omitempty"`
}

type appLifecycleCheckReport struct {
	Ready               bool     `json:"ready"`
	WhenReady           bool     `json:"whenReady"`
	EventOrder          []string `json:"eventOrder"`
	CancelledBeforeQuit bool     `json:"cancelledBeforeQuit"`
	QuitExitCode        int      `json:"quitExitCode"`
	Error               string   `json:"error,omitempty"`
}

type appWebAuthnCheckReport struct {
	Configured          bool   `json:"configured"`
	TouchIDSupported    bool   `json:"touchIDSupported"`
	KeychainGroupStored bool   `json:"keychainGroupStored"`
	InvalidRejected     bool   `json:"invalidRejected"`
	Error               string `json:"error,omitempty"`
}

type globalShortcutSuspensionCheckReport struct {
	InitialSuspended  bool   `json:"initialSuspended"`
	SuspendedAfterSet bool   `json:"suspendedAfterSet"`
	ResumedAfterUnset bool   `json:"resumedAfterUnset"`
	Error             string `json:"error,omitempty"`
}

type autoUpdaterCheckReport struct {
	Platform        string `json:"platform"`
	Supported       bool   `json:"supported"`
	FeedURL         string `json:"feedURL"`
	DefaultProvider string `json:"defaultProvider,omitempty"`
	Error           string `json:"error,omitempty"`
}

type menuCheckReport struct {
	ItemCount       int    `json:"itemCount"`
	FirstLabel      string `json:"firstLabel"`
	FirstEnabled    bool   `json:"firstEnabled"`
	CheckboxLabel   string `json:"checkboxLabel"`
	CheckboxChecked bool   `json:"checkboxChecked"`
	SubmenuFound    bool   `json:"submenuFound"`
	SubmenuRole     string `json:"submenuRole"`
	Error           string `json:"error,omitempty"`
}

type protocolCheckReport struct {
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

type sessionCheckReport struct {
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
	CacheClearResolved         bool   `json:"cacheClearResolved"`
	StorageClearResolved       bool   `json:"storageClearResolved"`
	Error                      string `json:"error,omitempty"`
}

type sessionQuotasCheckReport struct {
	AcceptsQuotas bool   `json:"acceptsQuotas"`
	ErrorName     string `json:"errorName"`
	Error         string `json:"error,omitempty"`
}

type sessionWebAuthnCheckReport struct {
	HandlerCalled      bool   `json:"handlerCalled"`
	RequestID          string `json:"requestId"`
	Origin             string `json:"origin"`
	CredentialCount    int    `json:"credentialCount"`
	SelectedCredential string `json:"selectedCredential"`
	Error              string `json:"error,omitempty"`
}

type utilityProcessCheckReport struct {
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

func clipboardReport() clipboardCheckReport {
	var clipboard egdesktop.Clipboard
	clipboard.WriteText("electron-go clipboard")
	text, err := clipboard.ReadText()
	if err != nil {
		return clipboardCheckReport{Error: err.Error()}
	}
	return clipboardCheckReport{
		TextRoundTrip: text == "electron-go clipboard",
		Text:          text,
	}
}

func clipboardFormatsReport() clipboardFormatsCheckReport {
	var clipboard egdesktop.Clipboard
	html := "<b>electron-go</b>"
	rtf := `{\\rtf1\\ansi electron-go}`

	clipboard.WriteHTML(html)
	clipboard.WriteRTF(rtf)
	if err := clipboard.WriteBuffer("electron-go/custom", []byte{4, 5, 6}); err != nil {
		return clipboardFormatsCheckReport{Error: err.Error()}
	}

	gotHTML, err := clipboard.ReadHTML()
	if err != nil {
		return clipboardFormatsCheckReport{Error: err.Error()}
	}
	gotRTF, err := clipboard.ReadRTF()
	if err != nil {
		return clipboardFormatsCheckReport{Error: err.Error()}
	}
	gotBuffer, err := clipboard.ReadBuffer("electron-go/custom")
	if err != nil {
		return clipboardFormatsCheckReport{Error: err.Error()}
	}

	return clipboardFormatsCheckReport{
		HTMLContainsPayload: strings.Contains(gotHTML, html),
		RTFRoundTrip:        gotRTF == rtf,
		RTFBytes:            len(gotRTF),
		BufferRoundTrip:     bytes.Equal(gotBuffer, []byte{4, 5, 6}),
		BufferBytes:         len(gotBuffer),
	}
}

func shellReport() shellCheckReport {
	var shell egdesktop.Shell
	message, err := shell.OpenPath(filepath.Join(os.TempDir(), "electron-go-shell-open-path-missing"))
	if err != nil {
		return shellCheckReport{Error: err.Error()}
	}
	return shellCheckReport{
		MissingPathRejected:  message != "",
		ErrorMessageNonempty: message != "",
	}
}

func contentTracingReport() contentTracingCheckReport {
	var tracer egdiagnostics.Tracer
	report := contentTracingCheckReport{
		Categories:    []string{"electron"},
		HeapProfiling: false,
	}
	if err := tracer.Start(egdiagnostics.TraceConfig{Categories: []string{"electron"}}); err != nil {
		report.Error = err.Error()
		return report
	}
	report.Started = true
	result, err := tracer.Stop("electron-go-trace.json")
	if err != nil {
		report.Error = err.Error()
		return report
	}
	report.Stopped = true
	report.RequestedPathReturned = result.Path == "electron-go-trace.json"
	report.Categories = result.Categories
	report.HeapProfiling = result.HeapProfile
	return report
}

func notificationReport() notificationCheckReport {
	notification, err := egnotification.New(egnotification.Options{
		Title:  "Build complete",
		Body:   "Artifacts are ready",
		Silent: true,
	})
	if err != nil {
		return notificationCheckReport{Error: err.Error()}
	}
	options := notification.Options()
	return notificationCheckReport{
		Title:          options.Title,
		Body:           options.Body,
		Silent:         options.Silent,
		DefaultUrgency: string(options.Urgency),
	}
}

func notificationIdentityReport() notificationIdentityCheckReport {
	platform := electronPlatform()
	report := notificationIdentityCheckReport{
		Platform:                   platform,
		Supported:                  platform == "darwin" || platform == "win32",
		HistoryAvailable:           platform == "darwin",
		RemoveFromHistoryAvailable: false,
	}
	if !report.Supported {
		return report
	}

	options := egnotification.Options{
		Title:   "Deploy complete",
		Body:    "Artifacts are ready",
		ID:      "deploy-42",
		GroupID: "deployments",
	}
	if platform == "win32" {
		options.GroupTitle = "Deployments"
	}
	notification, err := egnotification.New(options)
	if err != nil {
		report.Error = err.Error()
		return report
	}
	normalized := notification.Options()
	report.ID = normalized.ID
	report.GroupID = normalized.GroupID
	report.GroupTitle = normalized.GroupTitle
	report.Title = normalized.Title
	report.Body = normalized.Body
	return report
}

func electronPlatform() string {
	switch goruntime.GOOS {
	case "darwin":
		return "darwin"
	case "windows":
		return "win32"
	default:
		return goruntime.GOOS
	}
}

func browserWindowOptionsReport() browserWindowOptionsCheckReport {
	width := 640
	height := 480
	show := false
	devTools := false
	contextIsolation := false
	options, err := egbrowserwindow.NormalizeOptions(egbrowserwindow.ConstructorOptions{
		Width:    &width,
		Height:   &height,
		Show:     &show,
		ParentID: 42,
		Modal:    true,
		Title:    "Fixture",
		WebPreferences: egbrowserwindow.WebPreferences{
			DevTools:         &devTools,
			ContextIsolation: &contextIsolation,
			NodeIntegration:  true,
		},
	})
	if err != nil {
		return browserWindowOptionsCheckReport{Error: err.Error()}
	}
	defaultOptions, err := egbrowserwindow.NormalizeOptions(egbrowserwindow.ConstructorOptions{})
	if err != nil {
		return browserWindowOptionsCheckReport{Error: err.Error()}
	}
	window := egbrowserwindow.NewWindow(1, options)
	bounds := window.Bounds()
	return browserWindowOptionsCheckReport{
		Width:                    bounds.Width,
		Height:                   bounds.Height,
		Visible:                  window.IsVisible(),
		Title:                    window.Title(),
		DevTools:                 options.WebPreferences.DevTools,
		DefaultDevTools:          defaultOptions.WebPreferences.DevTools,
		DefaultContextIsolation:  defaultOptions.WebPreferences.ContextIsolation,
		ExplicitContextIsolation: options.WebPreferences.ContextIsolation,
		ExplicitNodeIntegration:  options.WebPreferences.NodeIntegration,
		ModalConstructed:         options.Modal && options.ParentID == 42,
	}
}

func browserWindowMethodsReport() browserWindowMethodsCheckReport {
	show := false
	options, err := egbrowserwindow.NormalizeOptions(egbrowserwindow.ConstructorOptions{Show: &show})
	if err != nil {
		return browserWindowMethodsCheckReport{Error: err.Error()}
	}
	window := egbrowserwindow.NewWindow(2, options)
	report := browserWindowMethodsCheckReport{
		InitialVisible: window.IsVisible(),
		EventOrder:     []string{},
	}
	if err := window.SetBounds(egbrowserwindow.Rect{X: 10, Y: 20, Width: 500, Height: 400}); err != nil {
		return browserWindowMethodsCheckReport{Error: err.Error()}
	}
	if err := window.SetSize(320, 240); err != nil {
		return browserWindowMethodsCheckReport{Error: err.Error()}
	}
	if err := window.Show(); err != nil {
		return browserWindowMethodsCheckReport{Error: err.Error()}
	}
	report.AfterShow = window.IsVisible()
	if err := window.Hide(); err != nil {
		return browserWindowMethodsCheckReport{Error: err.Error()}
	}
	report.AfterHide = window.IsVisible()
	if err := window.SetEnabled(false); err != nil {
		return browserWindowMethodsCheckReport{Error: err.Error()}
	}
	report.EnabledAfterDisable = window.IsEnabled()
	if err := window.SetEnabled(true); err != nil {
		return browserWindowMethodsCheckReport{Error: err.Error()}
	}
	report.EnabledAfterEnable = window.IsEnabled()
	if err := window.Show(); err != nil {
		return browserWindowMethodsCheckReport{Error: err.Error()}
	}
	report.EventOrder = append(report.EventOrder, "show")
	if err := window.Minimize(); err != nil {
		return browserWindowMethodsCheckReport{Error: err.Error()}
	}
	report.EventOrder = append(report.EventOrder, "minimize")
	report.MinimizedAfterMinimize = window.IsMinimized()
	if err := window.Restore(); err != nil {
		return browserWindowMethodsCheckReport{Error: err.Error()}
	}
	report.EventOrder = append(report.EventOrder, "restore")
	report.MinimizedAfterRestore = window.IsMinimized()
	if err := window.Maximize(); err != nil {
		return browserWindowMethodsCheckReport{Error: err.Error()}
	}
	report.EventOrder = append(report.EventOrder, "maximize")
	report.MaximizedAfterMaximize = window.IsMaximized()
	if err := window.Unmaximize(); err != nil {
		return browserWindowMethodsCheckReport{Error: err.Error()}
	}
	report.EventOrder = append(report.EventOrder, "unmaximize")
	report.MaximizedAfterUnmaximize = window.IsMaximized()
	shouldPreventClose := true
	window.On(egbrowserwindow.EventClose, func(ctx *egbrowserwindow.EventContext) {
		report.CloseEvent = true
		if shouldPreventClose {
			shouldPreventClose = false
			ctx.PreventDefault()
			report.ClosePrevented = true
			report.EventOrder = append(report.EventOrder, "close-prevented")
			return
		}
		report.EventOrder = append(report.EventOrder, "close")
	})
	window.On(egbrowserwindow.EventClosed, func(*egbrowserwindow.EventContext) {
		report.EventOrder = append(report.EventOrder, "closed")
		report.ClosedEvent = true
	})
	bounds := window.Bounds()
	normal := window.NormalBounds()
	report.Bounds = rectJSON{X: bounds.X, Y: bounds.Y, Width: bounds.Width, Height: bounds.Height}
	report.NormalBounds = rectJSON{X: normal.X, Y: normal.Y, Width: normal.Width, Height: normal.Height}
	if window.Close() {
		report.Error = "first close was not prevented"
		return report
	}
	report.UsableAfterPreventedClose = !window.IsClosed() && window.IsEnabled()
	report.Closed = window.Close()
	report.EnabledAfterEnd = window.IsEnabled()
	return report
}

func viewTreeReport() viewTreeCheckReport {
	root := egview.New(1)
	newParent := egview.New(5)
	a := egview.New(2)
	b := egview.New(3)
	c := egview.New(4)
	live := egview.NewWebContentsView(6, 60)
	report := viewTreeCheckReport{}
	c.On(egview.EventBoundsChanged, func(egview.Event) {
		report.BoundsChanged = true
	})
	if err := a.SetBounds(egview.Rect{Width: 100, Height: 20}, egview.SetBoundsOptions{}); err != nil {
		return viewTreeCheckReport{Error: err.Error()}
	}
	if err := b.SetBounds(egview.Rect{Width: 200, Height: 20}, egview.SetBoundsOptions{}); err != nil {
		return viewTreeCheckReport{Error: err.Error()}
	}
	if err := c.SetBounds(egview.Rect{X: 7, Y: 8, Width: 300, Height: 200}, egview.SetBoundsOptions{}); err != nil {
		return viewTreeCheckReport{Error: err.Error()}
	}
	if err := root.AddChild(a, nil); err != nil {
		return viewTreeCheckReport{Error: err.Error()}
	}
	if err := root.AddChild(b, nil); err != nil {
		return viewTreeCheckReport{Error: err.Error()}
	}
	index := 1
	if err := root.AddChild(c, &index); err != nil {
		return viewTreeCheckReport{Error: err.Error()}
	}
	report.RootChildWidths = childWidths(root)
	root.RemoveChild(b)
	report.AfterRemoveWidths = childWidths(root)
	if err := newParent.AddChild(a, nil); err != nil {
		return viewTreeCheckReport{Error: err.Error()}
	}
	report.AfterReparentWidths = childWidths(root)
	report.NewParentChildWidths = childWidths(newParent)
	bounds := c.Bounds()
	report.ProbeBounds = rectJSON{X: bounds.X, Y: bounds.Y, Width: bounds.Width, Height: bounds.Height}
	if err := live.SetBounds(egview.Rect{Width: 400, Height: 300}, egview.SetBoundsOptions{}); err != nil {
		return viewTreeCheckReport{Error: err.Error()}
	}
	if err := newParent.AddChild(live.View, nil); err != nil {
		return viewTreeCheckReport{Error: err.Error()}
	}
	report.LiveWebContentsParented = containsView(newParent.Children(), live.View)
	const liveURL = "data:text/html,<html><body>webcontentsview</body></html>"
	if err := live.WebContents().LoadURL(liveURL); err != nil {
		return viewTreeCheckReport{Error: err.Error()}
	}
	report.LiveWebContentsLoaded = strings.HasPrefix(live.WebContents().URL(), "data:text/html")
	if parsed, err := url.Parse(live.WebContents().URL()); err == nil {
		report.LiveWebContentsURL = parsed.Scheme + ":"
	}
	newParent.RemoveChild(live.View)
	report.LiveWebContentsDetached = !containsView(newParent.Children(), live.View)
	return report
}

func viewAnimationReport() viewAnimationCheckReport {
	v := egview.New(6)
	report := viewAnimationCheckReport{
		Animated:   true,
		DurationMS: 300,
		Easing:     "ease-in-out",
	}
	v.On(egview.EventBoundsChanged, func(egview.Event) {
		report.BoundsChanged = true
	})
	if err := v.SetBounds(
		egview.Rect{X: 0, Y: 0, Width: 300, Height: 200},
		egview.SetBoundsOptions{Animate: true, Duration: 300 * time.Millisecond},
	); err != nil {
		return viewAnimationCheckReport{Error: err.Error()}
	}
	bounds := v.Bounds()
	animation := v.LastAnimation()
	report.Animated = animation.Animate
	report.DurationMS = int(animation.Duration / time.Millisecond)
	report.Bounds = rectJSON{X: bounds.X, Y: bounds.Y, Width: bounds.Width, Height: bounds.Height}
	return report
}

func webContentsNavigationReport() webContentsNavigationCheckReport {
	wc := egwebcontents.New(8)
	report := webContentsNavigationCheckReport{}
	wc.On(egwebcontents.EventDidStartNavigation, func(*egwebcontents.EventContext) {
		report.DidStartNavigation = true
	})
	wc.On(egwebcontents.EventDidNavigate, func(*egwebcontents.EventContext) {
		report.DidNavigate = true
	})
	wc.On(egwebcontents.EventDidFinishLoad, func(*egwebcontents.EventContext) {
		report.DidFinishLoad = true
	})
	wc.On(egwebcontents.EventDidNavigateInPage, func(*egwebcontents.EventContext) {
		report.DidNavigateInPage = true
	})
	const baseURL = "data:text/html,<html><body>electron-go</body></html>"
	if err := wc.LoadURL(baseURL); err != nil {
		return webContentsNavigationCheckReport{Error: err.Error()}
	}
	report.CanGoBack = wc.CanGoBack()
	report.CanGoForward = wc.CanGoForward()
	report.LoadingAfter = wc.IsLoading()
	report.HistoryLength = len(wc.History())
	if parsed, err := url.Parse(wc.URL()); err == nil {
		report.FinalHash = parsed.Fragment
	}
	return report
}

func webContentsDevToolsTargetReport() webContentsDevToolsTargetCheckReport {
	first := egwebcontents.New(9)
	second := egwebcontents.New(10)
	firstID := first.GetOrCreateDevToolsTargetID()
	secondID := first.GetOrCreateDevToolsTargetID()
	otherID := second.GetOrCreateDevToolsTargetID()
	lookup, ok := egwebcontents.FromDevToolsTargetID(firstID)
	return webContentsDevToolsTargetCheckReport{
		MethodAvailable: true,
		FirstNonempty:   firstID != "",
		Stable:          firstID == secondID,
		Distinct:        firstID != otherID,
		Lookupable:      ok && lookup == first,
	}
}

func webContentsPrintReport() webContentsPrintCheckReport {
	wc := egwebcontents.New(11)
	job, err := wc.Print(egwebcontents.PrintOptions{UsePrinterDefaultPageSize: true})
	if err != nil {
		return webContentsPrintCheckReport{Error: err.Error()}
	}
	_, conflictErr := wc.Print(egwebcontents.PrintOptions{
		UsePrinterDefaultPageSize: true,
		PageSize:                  &egwebcontents.PageSize{WidthMicrons: 210000, HeightMicrons: 297000},
	})
	return webContentsPrintCheckReport{
		DefaultAccepted:  true,
		DefaultPageSize:  job.UsePrinterDefaultPageSize,
		PageSizeOmitted:  job.PageSize == nil,
		CopiesDefault:    job.Copies,
		ConflictRejected: errors.Is(conflictErr, egwebcontents.ErrInvalidPrintOptions),
	}
}

func focusOnNavigationReport() focusOnNavigationCheckReport {
	defaultOptions, err := egbrowserwindow.NormalizeOptions(egbrowserwindow.ConstructorOptions{})
	if err != nil {
		return focusOnNavigationCheckReport{Error: err.Error()}
	}
	focusOnNavigation := false
	explicitOptions, err := egbrowserwindow.NormalizeOptions(egbrowserwindow.ConstructorOptions{
		WebPreferences: egbrowserwindow.WebPreferences{FocusOnNavigation: &focusOnNavigation},
	})
	if err != nil {
		return focusOnNavigationCheckReport{Error: err.Error()}
	}
	return focusOnNavigationCheckReport{
		DefaultAccepted:       true,
		DefaultFocuses:        defaultOptions.WebPreferences.FocusOnNavigation,
		ExplicitFalseAccepted: true,
		ExplicitFalseFocuses:  explicitOptions.WebPreferences.FocusOnNavigation,
	}
}

func offscreenDeviceScaleReport() offscreenDeviceScaleCheckReport {
	defaultOptions, err := egbrowserwindow.NormalizeOptions(egbrowserwindow.ConstructorOptions{
		WebPreferences: egbrowserwindow.WebPreferences{
			Offscreen: egbrowserwindow.OffscreenPreferences{Enabled: true},
		},
	})
	if err != nil {
		return offscreenDeviceScaleCheckReport{Error: err.Error()}
	}
	scale := 2.0
	customOptions, err := egbrowserwindow.NormalizeOptions(egbrowserwindow.ConstructorOptions{
		WebPreferences: egbrowserwindow.WebPreferences{
			Offscreen: egbrowserwindow.OffscreenPreferences{
				Enabled:           true,
				DeviceScaleFactor: &scale,
			},
		},
	})
	if err != nil {
		return offscreenDeviceScaleCheckReport{Error: err.Error()}
	}
	return offscreenDeviceScaleCheckReport{
		DefaultAccepted:          defaultOptions.WebPreferences.Offscreen.Enabled,
		DefaultDeviceScaleFactor: defaultOptions.WebPreferences.Offscreen.DeviceScaleFactor,
		CustomAccepted:           customOptions.WebPreferences.Offscreen.Enabled,
		CustomDeviceScaleFactor:  customOptions.WebPreferences.Offscreen.DeviceScaleFactor,
	}
}

func ipcReport() ipcCheckReport {
	router := egipc.NewRouter()
	report := ipcCheckReport{}
	if err := router.Handle("fixture:ping", func(_ context.Context, msg egipc.Message) (egipc.Reply, error) {
		if len(msg.Args) == 1 && msg.Args[0] == "renderer" {
			report.ArgsEcho = true
		}
		return egipc.Reply{Value: "pong"}, nil
	}); err != nil {
		return ipcCheckReport{Error: err.Error()}
	}
	if err := router.Handle("fixture:ping", func(context.Context, egipc.Message) (egipc.Reply, error) {
		return egipc.Reply{}, nil
	}); errors.Is(err, egipc.ErrHandlerExists) {
		report.DuplicateRejected = true
	}
	onceCalls := 0
	if err := router.HandleOnce("fixture:once", func(context.Context, egipc.Message) (egipc.Reply, error) {
		onceCalls++
		return egipc.Reply{Value: onceCalls}, nil
	}); err != nil {
		return ipcCheckReport{Error: err.Error()}
	}
	if err := router.Handle("fixture:gone", func(context.Context, egipc.Message) (egipc.Reply, error) {
		return egipc.Reply{Value: "gone"}, nil
	}); err != nil {
		return ipcCheckReport{Error: err.Error()}
	}
	router.RemoveHandler("fixture:gone")
	reply, err := router.Invoke(context.Background(), egipc.Message{Channel: "fixture:ping", Args: []any{"renderer"}})
	if err != nil {
		return ipcCheckReport{Error: err.Error()}
	}
	report.InvokePong = reply.Value == "pong"
	reply, err = router.Invoke(context.Background(), egipc.Message{Channel: "fixture:once"})
	if err != nil {
		return ipcCheckReport{Error: err.Error()}
	}
	report.OnceFirst = reply.Value == 1
	_, err = router.Invoke(context.Background(), egipc.Message{Channel: "fixture:once"})
	report.OnceSecondRejected = errors.Is(err, egipc.ErrNoHandler)
	_, err = router.Invoke(context.Background(), egipc.Message{Channel: "fixture:gone"})
	report.RemovedRejected = errors.Is(err, egipc.ErrNoHandler)
	_, err = router.Invoke(context.Background(), egipc.Message{Channel: "fixture:missing"})
	report.MissingRejected = errors.Is(err, egipc.ErrNoHandler)
	port1, port2 := egipc.NewMessageChannel()
	port1.OnMessage(func(event egipc.MessageEvent) {
		report.MessagePortRoundTrip = event.Data == "from-port2"
	})
	if err := port2.PostMessage("from-port2"); err != nil {
		return ipcCheckReport{Error: err.Error()}
	}
	port1.Start()
	rendererPort, mainPort := egipc.NewMessageChannel()
	mainPort.OnMessage(func(event egipc.MessageEvent) {
		report.TransferredPortMessage = event.Data == "from-renderer-port"
	})
	mainPort.Start()
	if err := rendererPort.PostMessage("from-renderer-port"); err != nil {
		return ipcCheckReport{Error: err.Error()}
	}
	return report
}

func contextBridgeReport() contextBridgeCheckReport {
	bridge := egcontextbridge.New()
	api := egcontextbridge.Value{Kind: egcontextbridge.ValueObject, Object: map[string]egcontextbridge.Value{
		"version":           {Kind: egcontextbridge.ValueString, String: "1.0.0"},
		"add":               {Kind: egcontextbridge.ValueFunction, Function: "add"},
		"flags":             {Kind: egcontextbridge.ValueArray, Array: []egcontextbridge.Value{{Kind: egcontextbridge.ValueBool, Bool: true}}},
		"duplicateRejected": {Kind: egcontextbridge.ValueBool, Bool: true},
	}}
	if err := bridge.ExposeInMainWorld("fixture", api); err != nil {
		return contextBridgeCheckReport{Error: err.Error()}
	}
	duplicateErr := bridge.ExposeInMainWorld("fixture", egcontextbridge.Value{Kind: egcontextbridge.ValueNull})
	exposed, ok := bridge.Exposure(0, "fixture")
	if !ok {
		return contextBridgeCheckReport{Error: "fixture exposure missing"}
	}
	mutationErr := bridge.MutateExposure(0, "fixture", egcontextbridge.Value{Kind: egcontextbridge.ValueNull})
	return contextBridgeCheckReport{
		ExposedVersion:    exposed.Object["version"].String == "1.0.0",
		FunctionCallable:  exposed.Object["add"].Function == "add",
		ArrayValueCopied:  len(exposed.Object["flags"].Array) == 1 && exposed.Object["flags"].Array[0].Bool,
		DuplicateRejected: errors.Is(duplicateErr, egcontextbridge.ErrAlreadyExposed),
		MutationRejected:  errors.Is(mutationErr, egcontextbridge.ErrMutationNotAllowed),
	}
}

func childWidths(parent *egview.View) []int {
	children := parent.Children()
	widths := make([]int, 0, len(children))
	for _, child := range children {
		widths = append(widths, child.Bounds().Width)
	}
	return widths
}

func containsView(children []*egview.View, target *egview.View) bool {
	for _, child := range children {
		if child == target {
			return true
		}
	}
	return false
}

func nodeVersionsReport(target compat.TargetVersions) (nodeVersionsCheckReport, error) {
	v8Process := target.V8Process
	if v8Process == "" {
		v8Process = target.V8
	}
	versions, err := egnodecompat.ProcessVersions(egnodecompat.Versions{
		Electron: target.Electron,
		Chrome:   target.Chromium,
		Node:     target.Node,
		V8:       v8Process,
		Modules:  target.Modules,
	})
	if err != nil {
		return nodeVersionsCheckReport{}, err
	}
	return nodeVersionsCheckReport{
		Electron: versions["electron"],
		Chrome:   versions["chrome"],
		Node:     versions["node"],
		V8:       versions["v8"],
		Modules:  versions["modules"],
	}, nil
}

func globalShortcutSuspensionReport() globalShortcutSuspensionCheckReport {
	registry := egglobalshortcut.NewRegistry()
	report := globalShortcutSuspensionCheckReport{
		InitialSuspended: registry.IsSuspended(),
	}
	registry.SetSuspended(true)
	report.SuspendedAfterSet = registry.IsSuspended()
	registry.SetSuspended(false)
	report.ResumedAfterUnset = !registry.IsSuspended()
	return report
}

func appLifecycleReport() appLifecycleCheckReport {
	app := egapplifecycle.New()
	report := appLifecycleCheckReport{EventOrder: []string{}}
	app.WhenReady(func() {
		report.WhenReady = true
		report.EventOrder = append([]string{"when-ready"}, report.EventOrder...)
	})
	app.On(egapplifecycle.EventReady, func(*egapplifecycle.EventContext) {
		report.EventOrder = append(report.EventOrder, "ready")
	})
	app.On(egapplifecycle.EventBeforeQuit, func(ctx *egapplifecycle.EventContext) {
		if !report.CancelledBeforeQuit {
			ctx.PreventDefault()
			report.CancelledBeforeQuit = true
			report.EventOrder = append(report.EventOrder, "before-quit-cancelled")
			return
		}
		report.EventOrder = append(report.EventOrder, "before-quit")
		report.QuitExitCode = ctx.ExitCode
	})
	app.On(egapplifecycle.EventWillQuit, func(ctx *egapplifecycle.EventContext) {
		report.EventOrder = append(report.EventOrder, "will-quit")
		report.QuitExitCode = ctx.ExitCode
	})
	app.On(egapplifecycle.EventQuit, func(ctx *egapplifecycle.EventContext) {
		report.EventOrder = append(report.EventOrder, "quit")
		report.QuitExitCode = ctx.ExitCode
	})
	if err := app.MarkReady(); err != nil {
		report.Error = err.Error()
		return report
	}
	report.Ready = app.IsReady()
	if app.Quit(0) {
		report.Error = "first quit was not cancelled"
		return report
	}
	if !app.Quit(0) {
		report.Error = "second quit was cancelled"
	}
	return report
}

func menuReport() menuCheckReport {
	disabled := false
	built, err := egmenu.BuildFromTemplate([]egmenu.ItemTemplate{
		{ID: "open", Label: " Open ", Accelerator: "CmdOrCtrl+O"},
		{Type: egmenu.ItemSeparator},
		{ID: "toggle", Label: "Enabled", Type: egmenu.ItemCheckbox, Checked: true, Enabled: &disabled},
		{ID: "view", Label: "View", Submenu: []egmenu.ItemTemplate{{ID: "devtools", Role: egmenu.RoleToggleDevTools}}},
	})
	if err != nil {
		return menuCheckReport{Error: err.Error()}
	}
	items := built.Items()
	report := menuCheckReport{ItemCount: len(items)}
	if len(items) > 0 {
		report.FirstLabel = items[0].Label
		report.FirstEnabled = items[0].Enabled
	}
	if len(items) > 2 {
		report.CheckboxLabel = items[2].Label
		report.CheckboxChecked = items[2].Checked
	}
	if item, ok := built.ItemByID("devtools"); ok {
		report.SubmenuFound = true
		report.SubmenuRole = string(item.Role)
	}
	return report
}

func appWebAuthnReport() appWebAuthnCheckReport {
	var manager egwebauthn.Manager
	report := appWebAuthnCheckReport{
		TouchIDSupported: egwebauthn.SupportsTouchID(goruntime.GOOS),
	}
	if err := manager.Configure(egwebauthn.Config{
		TouchID: &egwebauthn.TouchIDConfig{KeychainAccessGroup: " TEAM.bundle "},
	}); err != nil {
		report.Error = err.Error()
		return report
	}
	report.Configured = true
	config := manager.Config()
	report.KeychainGroupStored = config.TouchID != nil && config.TouchID.KeychainAccessGroup == "TEAM.bundle"
	err := manager.Configure(egwebauthn.Config{
		TouchID: &egwebauthn.TouchIDConfig{KeychainAccessGroup: "bad\ngroup"},
	})
	report.InvalidRejected = errors.Is(err, egwebauthn.ErrInvalidConfig)
	if err != nil {
		report.Error = err.Error()
	}
	return report
}

func appPathsReport() appPathsCheckReport {
	wd, err := os.Getwd()
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	sessionDir, err := os.MkdirTemp("", "electron-go-session-*")
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	logsDir, err := os.MkdirTemp("", "electron-go-logs-*")
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}

	store := egapppaths.NewStore("ElectronGo", wd, egapppaths.Environment{})
	appPath := store.GetAppPath()
	homePath, err := store.GetPath(egapppaths.NameHome)
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	appDataPath, err := store.GetPath(egapppaths.NameAppData)
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	userDataPath, err := store.GetPath(egapppaths.NameUserData)
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	tempPath, err := store.GetPath(egapppaths.NameTemp)
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	exePath, err := store.GetPath(egapppaths.NameExe)
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	modulePath, err := store.GetPath(egapppaths.NameModule)
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	desktopPath, err := store.GetPath(egapppaths.NameDesktop)
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	documentsPath, err := store.GetPath(egapppaths.NameDocuments)
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	downloadsPath, err := store.GetPath(egapppaths.NameDownloads)
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	musicPath, err := store.GetPath(egapppaths.NameMusic)
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	picturesPath, err := store.GetPath(egapppaths.NamePictures)
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	videosPath, err := store.GetPath(egapppaths.NameVideos)
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	crashDumpsPath, err := store.GetPath(egapppaths.NameCrashDumps)
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	sessionDataPath, err := store.GetPath(egapppaths.NameSession)
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	if err := store.SetPath(egapppaths.NameSession, sessionDir); err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	sessionOverride, err := store.GetPath(egapppaths.NameSession)
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	if err := store.SetPath(egapppaths.NameLogs, logsDir); err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}
	logsOverride, err := store.GetPath(egapppaths.NameLogs)
	if err != nil {
		return appPathsCheckReport{Error: err.Error()}
	}

	return appPathsCheckReport{
		AppPathNonempty:             appPath != "",
		HomePathNonempty:            homePath != "",
		AppDataPathNonempty:         appDataPath != "",
		UserDataPathNonempty:        userDataPath != "",
		TempPathNonempty:            tempPath != "",
		ExePathNonempty:             exePath != "",
		ModulePathNonempty:          modulePath != "",
		DesktopPathNonempty:         desktopPath != "",
		DocumentsPathNonempty:       documentsPath != "",
		DownloadsPathNonempty:       downloadsPath != "",
		MusicPathNonempty:           musicPath != "",
		PicturesPathNonempty:        picturesPath != "",
		VideosPathNonempty:          videosPath != "",
		CrashDumpsPathNonempty:      crashDumpsPath != "",
		SessionDataDefaultsUserData: sessionDataPath == userDataPath,
		SessionDataOverrideWorks:    sessionOverride == sessionDir,
		LogsOverrideWorks:           logsOverride == logsDir,
	}
}

func asarReport() asarCheckReport {
	dir, err := os.MkdirTemp("", "electron-go-asar-")
	if err != nil {
		return asarCheckReport{Error: err.Error()}
	}
	defer os.RemoveAll(dir)
	archivePath := filepath.Join(dir, "app.asar")
	const mainSource = "console.log('asar')"
	const nativeSource = "native"
	if err := egasar.Write(archivePath, map[string][]byte{
		"main.js":           []byte(mainSource),
		"pkg/index.js":      []byte("module.exports = 42"),
		"native/addon.node": []byte(nativeSource),
	}, nil); err != nil {
		return asarCheckReport{Error: err.Error()}
	}
	archive, err := egasar.Open(archivePath)
	if err != nil {
		return asarCheckReport{Error: err.Error()}
	}
	main, err := archive.ReadFile("main.js")
	if err != nil {
		return asarCheckReport{Error: err.Error()}
	}
	index, err := archive.ResolveModule("pkg")
	if err != nil {
		return asarCheckReport{Error: err.Error()}
	}
	unpacked, err := archive.ReadFile("native/addon.node")
	if err != nil {
		return asarCheckReport{Error: err.Error()}
	}
	entry, err := archive.Stat("main.js")
	if err != nil {
		return asarCheckReport{Error: err.Error()}
	}
	return asarCheckReport{
		ReadMain:           string(main) == mainSource,
		RequireIndex:       index.Path == "pkg/index.js",
		UnpackedRead:       string(unpacked) == nativeSource,
		CopySourceUnpacked: false,
		StatSize:           entry.Size,
	}
}

func autoUpdaterReport() autoUpdaterCheckReport {
	platform := electronPlatform()
	report := autoUpdaterCheckReport{
		Platform:        platform,
		Supported:       platform == "darwin" || platform == "win32",
		DefaultProvider: string(egautoupdater.ProviderSquirrel),
	}
	if !report.Supported {
		return report
	}
	updater := egautoupdater.New("1.0.0")
	if err := updater.SetFeedURL(egautoupdater.Feed{
		URL:      "https://updates.example.test/feed",
		Provider: egautoupdater.ProviderSquirrel,
	}); err != nil {
		report.Error = err.Error()
		return report
	}
	report.FeedURL = updater.FeedURL().URL
	return report
}

func netLogReport() netLogCheckReport {
	var log egnet.NetLog
	report := netLogCheckReport{}
	if err := log.StartLogging("electron-go-netlog.json"); err != nil {
		report.Error = err.Error()
		return report
	}
	report.Started = true
	report.ActiveDuring = log.IsCurrentlyLogging()
	if _, err := log.StopLogging(); err != nil {
		report.Error = err.Error()
		return report
	}
	report.Stopped = true
	report.InactiveAfter = !log.IsCurrentlyLogging()
	return report
}

func clientRequestReport() clientRequestCheckReport {
	req, err := egnet.NewClientRequest(egnet.RequestOptions{
		Method:         " post ",
		URL:            "http://127.0.0.1/request?mode=manual",
		Headers:        http.Header{"X-Test": []string{"one"}},
		RedirectPolicy: egnet.RedirectManual,
		UploadBody:     &egnet.UploadBody{ContentType: "text/plain", Bytes: []byte("payload")},
	})
	if err != nil {
		return clientRequestCheckReport{Error: err.Error()}
	}
	req.End()
	body := ""
	if req.UploadBody != nil {
		body = string(req.UploadBody.Bytes)
	}
	parsedURL, err := req.ParsedURL()
	if err != nil {
		return clientRequestCheckReport{Error: err.Error()}
	}
	return clientRequestCheckReport{
		Method:         req.Method,
		Path:           parsedURL.Path,
		QueryMode:      parsedURL.Query().Get("mode"),
		Header:         req.Headers.Get("X-Test"),
		Body:           body,
		RedirectPolicy: string(req.RedirectPolicy),
		Ended:          req.Ended(),
	}
}

func protocolReport() protocolCheckReport {
	registry := egprotocol.NewRegistry()
	if err := registry.RegisterSchemesAsPrivileged([]egprotocol.SchemePrivilege{
		{
			Scheme: "egtest",
			Privileges: egprotocol.Privileges{
				Standard:        true,
				Secure:          true,
				SupportFetchAPI: true,
				AllowExtensions: true,
			},
		},
	}); err != nil {
		return protocolCheckReport{Error: err.Error()}
	}

	privileges, ok := registry.PrivilegesFor("egtest:")
	report := protocolCheckReport{
		RegisteredPrivileged: ok,
		AllowExtensions:      privileges.AllowExtensions,
	}

	if err := registry.RegisterHandler("egtest", egprotocol.Handler{
		Kind: egprotocol.HandlerString,
		Responder: func(request egprotocol.Request) (egprotocol.Response, error) {
			return egprotocol.Response{
				StatusCode: 201,
				Headers:    http.Header{"X-Eg-Protocol": []string{"handled"}, "Content-Type": []string{"text/plain"}},
				Body:       []byte("ok"),
			}, nil
		},
	}); err != nil {
		report.Error = err.Error()
		return report
	}
	report.HandledAfterRegister = registry.IsProtocolHandled("egtest")
	response, err := registry.Dispatch(egprotocol.Request{URL: "egtest://fixture/path?mode=fetch", Method: "GET"})
	if err != nil {
		report.Error = err.Error()
		return report
	}
	report.FetchStatus = response.StatusCode
	report.FetchHeader = response.Headers.Get("X-Eg-Protocol") == "handled"
	report.FetchBody = string(response.Body) == "ok"
	err = registry.RegisterHandler("egtest", egprotocol.Handler{Kind: egprotocol.HandlerString})
	report.DuplicateRejected = errors.Is(err, egprotocol.ErrAlreadyRegistered)
	if err != nil && !report.DuplicateRejected {
		report.Error = err.Error()
		return report
	}
	registry.LockPrivileges()
	err = registry.RegisterSchemesAsPrivileged([]egprotocol.SchemePrivilege{{Scheme: "late", Privileges: egprotocol.Privileges{Standard: true}}})
	report.LatePrivilegedRejected = errors.Is(err, egprotocol.ErrRegistryLocked)
	if err != nil && !report.LatePrivilegedRejected {
		report.Error = err.Error()
		return report
	}

	if err := registry.UnregisterProtocol("egtest:"); err != nil {
		report.Error = err.Error()
		return report
	}
	report.HandledAfterRemove = registry.IsProtocolHandled("egtest")
	return report
}

func sessionReport() sessionCheckReport {
	registry := egsession.NewRegistry()
	defaultSession := registry.DefaultSession()
	emptyPartition, err := registry.FromPartition("")
	if err != nil {
		return sessionCheckReport{Error: err.Error()}
	}
	persistent, err := registry.FromPartition("persist:egtest")
	if err != nil {
		return sessionCheckReport{Error: err.Error()}
	}
	inMemory, err := registry.FromPartition("egtest-memory")
	if err != nil {
		return sessionCheckReport{Error: err.Error()}
	}

	report := sessionCheckReport{
		DefaultSameWithEmpty:       defaultSession == emptyPartition,
		PersistStoragePathNonempty: persistent.IsPersistent(),
		MemoryStoragePathEmpty:     !inMemory.IsPersistent(),
	}

	if err := defaultSession.SetCookie(egsession.Cookie{
		URL:   "https://example.test/",
		Name:  "sid",
		Value: "1",
	}); err != nil {
		report.Error = err.Error()
		return report
	}
	if err := defaultSession.SetCookie(egsession.Cookie{
		URL:   "https://example.test/",
		Name:  "sid",
		Value: "2",
	}); err != nil {
		report.Error = err.Error()
		return report
	}
	cookies := defaultSession.Cookies()
	report.CookieCount = len(cookies)
	for _, cookie := range cookies {
		if cookie.Name == "sid" && cookie.Value == "2" {
			report.CookieRoundTrip = true
			break
		}
	}
	report.CookieOverwriteValue = report.CookieCount == 1 && report.CookieRoundTrip
	for _, change := range defaultSession.CookieChanges() {
		if change.Cookie.Name == "sid" && change.Cause == egsession.CookieChangeOverwrite && change.Removed {
			report.CookieOverwriteChange = true
			break
		}
	}
	if err := defaultSession.DeleteCookie("https://example.test/", "sid"); err != nil {
		report.Error = err.Error()
		return report
	}
	report.CookieRemoveResolved = true
	report.CookieRemoved = len(defaultSession.Cookies()) == 0
	for _, change := range defaultSession.CookieChanges() {
		if change.Cookie.Name == "sid" && change.Cause == egsession.CookieChangeExplicit && change.Removed {
			report.CookieRemoveChange = true
			break
		}
	}

	defaultSession.ClearCache()
	report.CacheClearResolved = defaultSession.CacheCleared()
	if err := defaultSession.ClearStorageData(egsession.ClearStorageOptions{
		Origins:  []string{"https://example.test"},
		Storages: []string{"cookies", "localstorage"},
	}); err != nil {
		report.Error = err.Error()
		return report
	}
	report.StorageClearResolved = len(defaultSession.StorageClears()) == 1
	return report
}

func sessionQuotasReport() sessionQuotasCheckReport {
	s := egsession.NewRegistry().DefaultSession()
	err := s.ClearStorageData(egsession.ClearStorageOptions{
		Quotas: []string{"temporary"},
	})
	report := sessionQuotasCheckReport{
		AcceptsQuotas: err == nil,
	}
	if err == nil {
		return report
	}
	if errors.Is(err, egsession.ErrUnsupportedClearOption) {
		report.ErrorName = "TypeError"
		return report
	}
	report.Error = err.Error()
	return report
}

func sessionWebAuthnReport() sessionWebAuthnCheckReport {
	s := egsession.NewRegistry().DefaultSession()
	report := sessionWebAuthnCheckReport{}
	s.SetWebAuthnAccountSelectionHandler(func(selection egsession.WebAuthnAccountSelection) (string, error) {
		report.HandlerCalled = true
		report.RequestID = selection.RequestID
		report.Origin = selection.Origin
		report.CredentialCount = len(selection.Credentials)
		return "credential-2", nil
	})
	selected, err := s.SelectWebAuthnAccount(egsession.WebAuthnAccountSelection{
		RequestID: " request ",
		Origin:    " https://example.test ",
		Credentials: []egsession.WebAuthnCredential{
			{ID: "credential-1", RelyingParty: "example.test", UserName: "one"},
			{ID: " credential-2 ", RelyingParty: "example.test", UserName: "two"},
		},
	})
	if err != nil {
		return sessionWebAuthnCheckReport{Error: err.Error()}
	}
	report.SelectedCredential = selected
	return report
}

func utilityProcessReport() utilityProcessCheckReport {
	process, err := egutilityprocess.Fork(egutilityprocess.Options{
		ModulePath:  "worker.js",
		Args:        []string{"--fixture"},
		Environment: []string{"FIXTURE_ENV=ok"},
		Stdio:       egutilityprocess.StdioPipe,
		ServiceName: "fixture-service",
		Sandbox:     true,
		DisclaimTCC: true,
	})
	if err != nil {
		return utilityProcessCheckReport{Error: err.Error()}
	}
	if err := process.Start(); err != nil {
		return utilityProcessCheckReport{Error: err.Error()}
	}
	if err := process.PostMessage([]byte("ready")); err != nil {
		return utilityProcessCheckReport{Error: err.Error()}
	}
	if err := process.Kill(0); err != nil {
		return utilityProcessCheckReport{Error: err.Error()}
	}

	options := process.Options()
	report := utilityProcessCheckReport{
		StdioMode:        string(options.Stdio),
		StdoutPiped:      options.Stdio == egutilityprocess.StdioPipe,
		StderrPiped:      options.Stdio == egutilityprocess.StdioPipe,
		StdinPiped:       false,
		StdinWrite:       false,
		Args:             options.Args,
		ServiceName:      options.ServiceName,
		EnvironmentValue: "",
	}
	for _, env := range options.Environment {
		if strings.HasPrefix(env, "FIXTURE_ENV=") {
			report.EnvironmentValue = strings.TrimPrefix(env, "FIXTURE_ENV=")
		}
	}
	for _, event := range process.Events() {
		switch event.Event {
		case egutilityprocess.EventSpawn:
			report.SpawnEvent = true
		case egutilityprocess.EventMessage:
			report.MessageEvent = string(event.Message) == "ready"
		case egutilityprocess.EventExit:
			report.ExitEvent = true
			report.ExitCode = event.ExitCode
		}
	}
	return report
}

func safeStorageReport(ctx context.Context) safeStorageCheckReport {
	storage, err := egstorage.New(egstorage.BackendBasicText, true, nil)
	if err != nil {
		return safeStorageCheckReport{Error: err.Error()}
	}

	report := safeStorageCheckReport{
		Available: storage.IsEncryptionAvailable(),
		Backend:   string(storage.Backend()),
	}

	available, err := storage.IsEncryptionAvailableAsync(ctx)
	if err != nil {
		report.Error = err.Error()
		return report
	}
	report.AsyncAvailable = available

	ciphertext, err := storage.EncryptStringAsync(ctx, "token")
	if err != nil {
		report.Error = err.Error()
		return report
	}
	report.CiphertextBytes = len(ciphertext)

	plain, err := storage.DecryptStringAsync(ctx, ciphertext)
	if err != nil {
		report.Error = err.Error()
		return report
	}
	report.RoundTrip = plain == "token"
	return report
}

func findHelloFixtureDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return helloFixtureDir
	}
	for {
		candidate := filepath.Join(wd, helloFixtureDir)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return helloFixtureDir
		}
		wd = parent
	}
}
