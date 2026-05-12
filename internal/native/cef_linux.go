//go:build linux && amd64 && cgo && electron_go_cef

package native

/*
#cgo CFLAGS: -DELECTRON_GO_HAS_CEF -I${SRCDIR}/../../native -I${SRCDIR}/../../native/cef/current
#cgo LDFLAGS: -L${SRCDIR}/../../bin -lcef -Wl,-rpath,$ORIGIN
#include <stdlib.h>
#include "cef_shim.h"
*/
import "C"

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

var cefContextInitialized atomic.Bool
var cefLastBrowserID atomic.Int64
var cefLastHTTPStatus atomic.Int32
var cefLastLoadError atomic.Int32
var cefBrowserClosed atomic.Bool
var cefLoadEndUnixNano atomic.Int64
var cefLoadErrorUnixNano atomic.Int64
var cefBeforeCloseUnixNano atomic.Int64

var cefBrowserProcessSwitches = []string{
	"--disable-background-networking",
	"--disable-breakpad",
	"--disable-component-update",
	"--disable-default-apps",
	"--disable-extensions",
	"--disable-features=AutofillServerCommunication,CertificateTransparencyComponentUpdater,MediaRouter,OptimizationHints",
	"--disable-gpu",
	"--disable-gpu-sandbox",
	"--disable-dev-shm-usage",
	"--disable-sync",
	"--in-process-gpu",
	"--metrics-recording-only",
	"--no-default-browser-check",
	"--no-first-run",
	"--no-sandbox",
	"--no-zygote",
}

type CEFBridge struct{}

func NewBridge() Bridge {
	return CEFBridge{}
}

func (b CEFBridge) InitializeForStart(ctx context.Context, req StartRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateStartRequest(req); err != nil {
		return err
	}
	return b.InitializeValidatedForStart(ctx, req)
}

func (b CEFBridge) InitializeValidatedForStart(ctx context.Context, req StartRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	initReq, err := NewCEFInitializeRequest(req.AppDir, req.Args)
	if err != nil {
		return err
	}
	return initializeCEF(ctx, initReq)
}

func (b CEFBridge) Shutdown(ctx context.Context) error {
	return shutdownCEF(ctx)
}

func (b CEFBridge) CreateBrowserWindow(ctx context.Context, req BrowserWindowCreateRequest) (int64, error) {
	if err := createBrowserWindow(ctx, req); err != nil {
		return 0, err
	}
	return cefLastBrowserID.Load(), nil
}

func (b CEFBridge) LoadURL(ctx context.Context, req BrowserWindowLoadRequest) error {
	if err := loadBrowserWindowURL(ctx, req); err != nil {
		return err
	}
	return runMessageLoopUntilLoad(ctx)
}

func (b CEFBridge) CloseBrowserWindow(ctx context.Context, req BrowserWindowCloseRequest) error {
	if cefBrowserClosed.Load() && cefLastBrowserID.Load() == req.BrowserID {
		return nil
	}
	if err := closeBrowserWindow(ctx, req); err != nil {
		return err
	}
	return runMessageLoop(ctx)
}

func (b CEFBridge) SetLoadEndScript(ctx context.Context, req BrowserWindowScriptRequest) error {
	return setBrowserWindowLoadEndScript(ctx, req)
}

func (b CEFBridge) WaitForLoad(ctx context.Context) error {
	return runMessageLoopUntilLoad(ctx)
}

func (b CEFBridge) Start(ctx context.Context, req StartRequest) (*StartResult, error) {
	traceStart := time.Now()
	trace := map[string]int64{}
	mark := func(stage string, started time.Time) {
		trace[stage] = time.Since(started).Milliseconds()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateStartRequest(req); err != nil {
		return nil, err
	}
	initReq, err := NewCEFInitializeRequest(req.AppDir, req.Args)
	if err != nil {
		return nil, err
	}
	stageStart := time.Now()
	if err := initializeCEF(ctx, initReq); err != nil {
		return nil, err
	}
	mark("cef_initialize", stageStart)
	stageStart = time.Now()
	loadURL, err := appIndexFileURL(req.AppDir)
	if err != nil {
		_ = shutdownCEF(ctx)
		return nil, err
	}
	mark("resolve_app_url", stageStart)
	stageStart = time.Now()
	if err := createBrowserWindow(ctx, BrowserWindowCreateRequest{
		ABIRevision:     CurrentABIRevision,
		URL:             loadURL,
		Width:           800,
		Height:          600,
		Show:            true,
		AutoCloseOnLoad: true,
	}); err != nil {
		_ = shutdownCEF(ctx)
		return nil, err
	}
	mark("create_browser", stageStart)
	stageStart = time.Now()
	if err := runMessageLoop(ctx); err != nil {
		_ = shutdownCEF(ctx)
		return nil, err
	}
	mark("message_loop", stageStart)
	addMessageLoopTrace(trace, stageStart, time.Now())
	stageStart = time.Now()
	if err := shutdownCEF(ctx); err != nil {
		return nil, err
	}
	mark("cef_shutdown", stageStart)
	trace["total_native_start"] = time.Since(traceStart).Milliseconds()
	return &StartResult{
		PID:            os.Getpid(),
		WindowCount:    1,
		Status:         StatusStopped,
		BridgeRevision: fmt.Sprintf("abi-%d-cef", CurrentABIRevision),
		Platform:       runtime.GOOS,
		StartupTraceMS: trace,
	}, nil
}

func ExecuteCEFSubprocess(ctx context.Context, args []string) (SubprocessExecutionResult, bool, error) {
	if err := ctx.Err(); err != nil {
		return SubprocessExecutionResult{}, false, err
	}
	cefArgs := appendCEFBrowserProcessSwitches(args)
	argv, err := newCStringViews(cefArgs)
	if err != nil {
		return SubprocessExecutionResult{}, false, err
	}
	defer argv.free()

	req := C.eg_cef_execute_process_request{
		abi_revision: C.uint32_t(CurrentABIRevision),
		argc:         C.uint64_t(len(cefArgs)),
		argv:         argv.ptr(),
	}
	out := C.eg_cef_subprocess_result{}

	status := C.eg_cef_shim_execute_process(&req, &out)
	result := SubprocessExecutionResult{
		ExitCode: int(out.exit_code),
		Status:   nativeStatus(out.status),
	}
	if status == C.EG_BRIDGE_STATUS_INVALID_REQUEST || status == C.EG_BRIDGE_STATUS_FAILED {
		return result, false, fmt.Errorf("cef_execute_process failed: status=%s", bridgeStatusName(status))
	}
	return result, result.ExitCode >= 0, nil
}

func CheckCEFInitialize(ctx context.Context, appDir string, args []string) error {
	req, err := NewCEFInitializeRequest(appDir, args)
	if err != nil {
		return err
	}
	if err := initializeCEF(ctx, req); err != nil {
		return err
	}
	return shutdownCEF(ctx)
}

func CheckRuntimeProcessModel(ctx context.Context, appDir string, args []string) (RuntimeProcessModelReport, error) {
	report := RuntimeProcessModelReport{
		PID:       os.Getpid(),
		NoSandbox: true,
		RemainingKnownGaps: []string{
			"CEF currently runs with no_sandbox=1 in CI; sandbox hardening remains gated separately.",
			"Electron utilityProcess semantics remain gated by utility-process-tcc-disclaim-electron-42.",
			"Crash reporter and Node process APIs remain gated by their dedicated ledger items.",
		},
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	report.MainThreadIDBefore = currentOSThreadID()

	req, err := NewCEFInitializeRequest(appDir, args)
	if err != nil {
		return report, err
	}
	loadURL, err := appIndexFileURL(appDir)
	if err != nil {
		return report, err
	}

	cefContextInitialized.Store(false)
	cefLastBrowserID.Store(0)
	cefLastHTTPStatus.Store(0)
	cefLastLoadError.Store(0)
	cefBrowserClosed.Store(false)
	resetCEFCallbackTrace()

	sampler := startProcessModelSampler(os.Getpid(), 2*time.Millisecond)
	initialized := false
	var lifecycleErr error
	if err := initializeCEF(ctx, req); err != nil {
		lifecycleErr = err
	} else {
		initialized = true
		if err := createBrowserWindow(ctx, BrowserWindowCreateRequest{
			ABIRevision:     CurrentABIRevision,
			URL:             loadURL,
			Width:           800,
			Height:          600,
			Show:            true,
			AutoCloseOnLoad: true,
		}); err != nil {
			lifecycleErr = err
		} else if err := runMessageLoop(ctx); err != nil {
			lifecycleErr = err
		}
	}
	if initialized {
		if err := shutdownCEF(ctx); lifecycleErr == nil && err != nil {
			lifecycleErr = err
		}
	}

	report.ObservedDescendants = sampler.stopAndReport()
	report.ObservedProcessTypes = ProcessTypesFromObservations(report.ObservedDescendants)
	reapExitedChildren(500 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	if zombies, err := listDescendantProcesses(os.Getpid()); err == nil {
		report.ZombieDescendants = zombies
		report.NoZombieDescendants = len(zombies) == 0
	}
	report.MainThreadIDAfter = currentOSThreadID()
	report.MainThreadStable = report.MainThreadIDBefore > 0 && report.MainThreadIDBefore == report.MainThreadIDAfter
	report.ContextInitialized = cefContextInitialized.Load()
	report.BrowserID = cefLastBrowserID.Load()
	report.MainFrameLoaded = report.BrowserID > 0 && cefLastLoadError.Load() == 0
	report.WindowClosed = cefBrowserClosed.Load()

	if lifecycleErr != nil {
		return report, lifecycleErr
	}
	if err := ValidateRuntimeProcessModelReport(report); err != nil {
		return report, err
	}
	return report, nil
}

func NewCEFInitializeRequest(appDir string, args []string) (CEFInitializeRequest, error) {
	if appDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return CEFInitializeRequest{}, err
		}
		appDir = wd
	}
	absAppDir, err := filepath.Abs(appDir)
	if err != nil {
		return CEFInitializeRequest{}, err
	}
	cachePath := cefCachePath(absAppDir)
	if err := ensureCEFCachePath(cachePath); err != nil {
		return CEFInitializeRequest{}, err
	}
	return NormalizeCEFInitializeRequest(CEFInitializeRequest{
		AppDir: absAppDir,
		Args:   appendCEFBrowserProcessSwitches(args),
		Settings: CEFSettings{
			NoSandbox:   true,
			CachePath:   cachePath,
			LogSeverity: CEFLogSeverityError,
		},
	}), nil
}

func ensureCEFCachePath(path string) error {
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			return nil
		}
		return fmt.Errorf("CEF cache path is not a directory: %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(path, 0o755)
}

func cefCachePath(absAppDir string) string {
	root, err := os.UserCacheDir()
	if err != nil || root == "" {
		root = os.TempDir()
	}
	sum := sha256.Sum256([]byte(absAppDir))
	return filepath.Join(root, "electron-go", hex.EncodeToString(sum[:8]), "cef")
}

func appendCEFBrowserProcessSwitches(args []string) []string {
	out := make([]string, 0, len(args)+len(cefBrowserProcessSwitches)+1)
	out = append(out, args...)
	if IsCEFSubprocessArgs(out) {
		return out
	}
	for _, flag := range cefBrowserProcessSwitches {
		if !hasSwitch(out, flag) {
			out = append(out, flag)
		}
	}
	if !hasSwitch(out, "--browser-subprocess-path") {
		if executable, err := os.Executable(); err == nil && executable != "" {
			out = append(out, "--browser-subprocess-path="+executable)
		}
	}
	return out
}

func hasSwitch(args []string, name string) bool {
	for _, arg := range args {
		if arg == name || strings.HasPrefix(arg, name+"=") {
			return true
		}
	}
	return false
}

func initializeCEF(ctx context.Context, req CEFInitializeRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	req = NormalizeCEFInitializeRequest(req)
	if err := ValidateCEFInitializeRequest(req); err != nil {
		return err
	}

	argv, err := newCStringViews(req.Args)
	if err != nil {
		return err
	}
	defer argv.free()
	appDir := newCStringView(req.AppDir)
	defer appDir.free()
	cachePath := newCStringView(req.Settings.CachePath)
	defer cachePath.free()

	cReq := C.eg_cef_initialize_request{
		abi_revision: C.uint32_t(req.ABIRevision),
		app_dir:      appDir.view,
		argc:         C.uint64_t(len(req.Args)),
		argv:         argv.ptr(),
	}
	cReq.settings.no_sandbox = boolToCUint8(req.Settings.NoSandbox)
	cReq.settings.cache_path = cachePath.view
	cReq.settings.log_severity = cLogSeverity(req.Settings.LogSeverity)

	status := C.eg_cef_shim_initialize(nil, &cReq)
	if status != C.EG_BRIDGE_STATUS_RUNNING {
		return fmt.Errorf("cef_initialize returned 0: status=%s", bridgeStatusName(status))
	}
	return nil
}

func shutdownCEF(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	status := C.eg_cef_shim_shutdown(nil)
	if status != C.EG_BRIDGE_STATUS_STOPPED {
		return fmt.Errorf("cef_shutdown failed: status=%s", bridgeStatusName(status))
	}
	return nil
}

func createBrowserWindow(ctx context.Context, req BrowserWindowCreateRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	req = NormalizeBrowserWindowCreateRequest(req)
	if err := ValidateBrowserWindowCreateRequest(req); err != nil {
		return err
	}
	cefLastBrowserID.Store(0)
	cefLastHTTPStatus.Store(0)
	cefLastLoadError.Store(0)
	cefBrowserClosed.Store(false)
	resetCEFCallbackTrace()

	urlView := newCStringView(req.URL)
	defer urlView.free()

	cReq := C.eg_browser_window_create_request{
		abi_revision:       C.uint32_t(req.ABIRevision),
		url:                urlView.view,
		width:              C.int32_t(req.Width),
		height:             C.int32_t(req.Height),
		show:               boolToCUint8(req.Show),
		auto_close_on_load: boolToCUint8(req.AutoCloseOnLoad),
	}
	out := C.eg_browser_window_result{}

	status := C.eg_cef_shim_create_browser_sync(nil, &cReq, &out)
	if status != C.EG_BRIDGE_STATUS_RUNNING {
		return fmt.Errorf("CEF BrowserWindow create failed: status=%s", bridgeStatusName(status))
	}
	return nil
}

func loadBrowserWindowURL(ctx context.Context, req BrowserWindowLoadRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	req = NormalizeBrowserWindowLoadRequest(req)
	if err := ValidateBrowserWindowLoadRequest(req); err != nil {
		return err
	}

	urlView := newCStringView(req.URL)
	defer urlView.free()

	cReq := C.eg_browser_window_load_request{
		abi_revision: C.uint32_t(req.ABIRevision),
		browser_id:   C.int64_t(req.BrowserID),
		url:          urlView.view,
	}

	status := C.eg_cef_shim_load_url(nil, &cReq)
	if status != C.EG_BRIDGE_STATUS_RUNNING {
		return fmt.Errorf("CEF BrowserWindow loadURL failed: status=%s", bridgeStatusName(status))
	}
	return nil
}

func closeBrowserWindow(ctx context.Context, req BrowserWindowCloseRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	req = NormalizeBrowserWindowCloseRequest(req)
	if err := ValidateBrowserWindowCloseRequest(req); err != nil {
		return err
	}

	cReq := C.eg_browser_window_close_request{
		abi_revision: C.uint32_t(req.ABIRevision),
		browser_id:   C.int64_t(req.BrowserID),
	}

	status := C.eg_cef_shim_close_browser(nil, &cReq)
	if status != C.EG_BRIDGE_STATUS_RUNNING {
		return fmt.Errorf("CEF BrowserWindow close failed: status=%s", bridgeStatusName(status))
	}
	return nil
}

func setBrowserWindowLoadEndScript(ctx context.Context, req BrowserWindowScriptRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	req = NormalizeBrowserWindowScriptRequest(req)
	if err := ValidateBrowserWindowScriptRequest(req); err != nil {
		return err
	}

	scriptView := newCStringView(req.Script)
	defer scriptView.free()

	cReq := C.eg_browser_window_script_request{
		abi_revision: C.uint32_t(req.ABIRevision),
		browser_id:   C.int64_t(req.BrowserID),
		script:       scriptView.view,
	}

	status := C.eg_cef_shim_set_load_end_script(nil, &cReq)
	if status != C.EG_BRIDGE_STATUS_RUNNING {
		return fmt.Errorf("CEF BrowserWindow set load-end script failed: status=%s", bridgeStatusName(status))
	}
	return nil
}

func runMessageLoop(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	status := C.eg_cef_shim_run_message_loop(nil)
	if status == C.EG_BRIDGE_STATUS_FAILED {
		return fmt.Errorf("CEF BrowserWindow lifecycle failed: browser_id=%d status=%d error=%d closed=%t", cefLastBrowserID.Load(), cefLastHTTPStatus.Load(), cefLastLoadError.Load(), cefBrowserClosed.Load())
	}
	if status != C.EG_BRIDGE_STATUS_STOPPED {
		return fmt.Errorf("CEF message loop failed: status=%s", bridgeStatusName(status))
	}
	return nil
}

func runMessageLoopUntilLoad(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	stageStart := time.Now()
	status := C.eg_cef_shim_run_message_loop_until_load(nil)
	if status == C.EG_BRIDGE_STATUS_FAILED {
		return fmt.Errorf("CEF BrowserWindow load failed: browser_id=%d status=%d error=%d elapsed_ms=%d", cefLastBrowserID.Load(), cefLastHTTPStatus.Load(), cefLastLoadError.Load(), time.Since(stageStart).Milliseconds())
	}
	if status != C.EG_BRIDGE_STATUS_STOPPED {
		return fmt.Errorf("CEF message loop load wait failed: status=%s", bridgeStatusName(status))
	}
	return nil
}

func appIndexFileURL(appDir string) (string, error) {
	indexPath := filepath.Join(appDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return "", fmt.Errorf("locate BrowserWindow fixture: %w", err)
	}
	abs, err := filepath.Abs(indexPath)
	if err != nil {
		return "", err
	}
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}).String(), nil
}

type cStringView struct {
	view C.eg_string_view
	ptr  *C.char
}

func newCStringView(value string) cStringView {
	ptr := C.CString(value)
	return cStringView{
		view: C.eg_string_view{
			data: ptr,
			len:  C.uint64_t(len(value)),
		},
		ptr: ptr,
	}
}

func (v cStringView) free() {
	if v.ptr != nil {
		C.free(unsafe.Pointer(v.ptr))
	}
}

type cStringViews struct {
	views *C.eg_string_view
	len   int
	ptrs  []*C.char
}

func newCStringViews(values []string) (*cStringViews, error) {
	out := &cStringViews{
		len:  len(values),
		ptrs: make([]*C.char, len(values)),
	}
	if len(values) > 0 {
		out.views = (*C.eg_string_view)(C.calloc(
			C.size_t(len(values)),
			C.size_t(unsafe.Sizeof(C.eg_string_view{})),
		))
		if out.views == nil {
			return nil, fmt.Errorf("allocate C argv string views")
		}
	}
	views := unsafe.Slice(out.views, len(values))
	for i, value := range values {
		ptr := C.CString(value)
		if ptr == nil {
			out.free()
			return nil, fmt.Errorf("allocate C string for argv[%d]", i)
		}
		views[i] = C.eg_string_view{
			data: ptr,
			len:  C.uint64_t(len(value)),
		}
		out.ptrs[i] = ptr
	}
	return out, nil
}

func (v *cStringViews) ptr() *C.eg_string_view {
	if v == nil || v.len == 0 {
		return nil
	}
	return v.views
}

func (v *cStringViews) free() {
	if v == nil {
		return
	}
	for _, ptr := range v.ptrs {
		if ptr != nil {
			C.free(unsafe.Pointer(ptr))
		}
	}
	if v.views != nil {
		C.free(unsafe.Pointer(v.views))
		v.views = nil
	}
}

func boolToCUint8(value bool) C.uint8_t {
	if value {
		return 1
	}
	return 0
}

func cLogSeverity(severity CEFLogSeverity) C.eg_cef_log_severity {
	switch severity {
	case CEFLogSeverityVerbose:
		return C.EG_CEF_LOG_SEVERITY_VERBOSE
	case CEFLogSeverityInfo:
		return C.EG_CEF_LOG_SEVERITY_INFO
	case CEFLogSeverityWarning:
		return C.EG_CEF_LOG_SEVERITY_WARNING
	case CEFLogSeverityError:
		return C.EG_CEF_LOG_SEVERITY_ERROR
	case CEFLogSeverityFatal:
		return C.EG_CEF_LOG_SEVERITY_FATAL
	case CEFLogSeverityDisable:
		return C.EG_CEF_LOG_SEVERITY_DISABLE
	default:
		return C.EG_CEF_LOG_SEVERITY_DEFAULT
	}
}

func nativeStatus(status C.eg_bridge_status) Status {
	switch status {
	case C.EG_BRIDGE_STATUS_UNAVAILABLE:
		return StatusUnavailable
	case C.EG_BRIDGE_STATUS_INVALID_REQUEST:
		return StatusInvalidRequest
	case C.EG_BRIDGE_STATUS_STARTING:
		return StatusStarting
	case C.EG_BRIDGE_STATUS_RUNNING:
		return StatusRunning
	case C.EG_BRIDGE_STATUS_STOPPED:
		return StatusStopped
	default:
		return StatusUnavailable
	}
}

func bridgeStatusName(status C.eg_bridge_status) string {
	switch status {
	case C.EG_BRIDGE_STATUS_UNAVAILABLE:
		return "unavailable"
	case C.EG_BRIDGE_STATUS_INVALID_REQUEST:
		return "invalid-request"
	case C.EG_BRIDGE_STATUS_STARTING:
		return "starting"
	case C.EG_BRIDGE_STATUS_RUNNING:
		return "running"
	case C.EG_BRIDGE_STATUS_STOPPED:
		return "stopped"
	case C.EG_BRIDGE_STATUS_FAILED:
		return "failed"
	default:
		return fmt.Sprintf("unknown(%d)", int(status))
	}
}

func resetCEFCallbackTrace() {
	cefLoadEndUnixNano.Store(0)
	cefLoadErrorUnixNano.Store(0)
	cefBeforeCloseUnixNano.Store(0)
}

func addMessageLoopTrace(trace map[string]int64, loopStart, loopEnd time.Time) {
	loadEnd := cefLoadEndUnixNano.Load()
	loadError := cefLoadErrorUnixNano.Load()
	beforeClose := cefBeforeCloseUnixNano.Load()
	if loadEnd > 0 {
		trace["message_loop_to_load_end"] = time.Unix(0, loadEnd).Sub(loopStart).Milliseconds()
	}
	if loadError > 0 {
		trace["message_loop_to_load_error"] = time.Unix(0, loadError).Sub(loopStart).Milliseconds()
	}
	if loadEnd > 0 && beforeClose > 0 {
		trace["load_end_to_before_close"] = time.Unix(0, beforeClose).Sub(time.Unix(0, loadEnd)).Milliseconds()
	}
	if beforeClose > 0 {
		trace["before_close_to_loop_return"] = loopEnd.Sub(time.Unix(0, beforeClose)).Milliseconds()
	}
}

//export goOnContextInitialized
func goOnContextInitialized() {
	cefContextInitialized.Store(true)
}

//export goOnBrowserAfterCreated
func goOnBrowserAfterCreated(browserID C.int) {
	cefLastBrowserID.Store(int64(browserID))
}

//export goOnBrowserLoadEnd
func goOnBrowserLoadEnd(browserID C.int, httpStatusCode C.int) {
	cefLastBrowserID.Store(int64(browserID))
	cefLastHTTPStatus.Store(int32(httpStatusCode))
	cefLastLoadError.Store(0)
	cefLoadEndUnixNano.Store(time.Now().UnixNano())
}

//export goOnBrowserLoadError
func goOnBrowserLoadError(browserID C.int, errorCode C.int) {
	cefLastBrowserID.Store(int64(browserID))
	cefLastLoadError.Store(int32(errorCode))
	cefLoadErrorUnixNano.Store(time.Now().UnixNano())
}

//export goOnBrowserBeforeClose
func goOnBrowserBeforeClose(browserID C.int) {
	cefLastBrowserID.Store(int64(browserID))
	cefBrowserClosed.Store(true)
	cefBeforeCloseUnixNano.Store(time.Now().UnixNano())
}
