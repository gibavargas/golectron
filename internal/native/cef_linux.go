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
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"unsafe"
)

var cefContextInitialized atomic.Bool
var cefLastBrowserID atomic.Int64
var cefLastHTTPStatus atomic.Int32
var cefLastLoadError atomic.Int32
var cefBrowserClosed atomic.Bool

var cefBrowserProcessSwitches = []string{
	"--disable-gpu",
	"--disable-gpu-compositing",
	"--disable-gpu-sandbox",
	"--disable-dev-shm-usage",
	"--in-process-gpu",
	"--no-sandbox",
}

type CEFBridge struct{}

func NewBridge() Bridge {
	return CEFBridge{}
}

func (b CEFBridge) Start(ctx context.Context, req StartRequest) (*StartResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateStartRequest(req); err != nil {
		return nil, err
	}
	if hash := C.eg_cef_shim_link_proof(); hash == nil {
		return nil, fmt.Errorf("CEF link proof failed")
	}
	initReq, err := NewCEFInitializeRequest(req.AppDir, req.Args)
	if err != nil {
		return nil, err
	}
	if err := initializeCEF(ctx, initReq); err != nil {
		return nil, err
	}
	loadURL, err := appIndexFileURL(req.AppDir)
	if err != nil {
		_ = shutdownCEF(ctx)
		return nil, err
	}
	if err := createBrowserWindow(ctx, BrowserWindowCreateRequest{
		ABIRevision: CurrentABIRevision,
		URL:         loadURL,
		Width:       800,
		Height:      600,
		Show:        true,
	}); err != nil {
		_ = shutdownCEF(ctx)
		return nil, err
	}
	if err := runMessageLoop(ctx); err != nil {
		_ = shutdownCEF(ctx)
		return nil, err
	}
	if err := shutdownCEF(ctx); err != nil {
		return nil, err
	}
	return &StartResult{
		PID:            os.Getpid(),
		WindowCount:    1,
		Status:         StatusStopped,
		BridgeRevision: fmt.Sprintf("abi-%d-cef", CurrentABIRevision),
		Platform:       runtime.GOOS,
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

	req := (*C.eg_cef_execute_process_request)(C.calloc(1, C.size_t(unsafe.Sizeof(C.eg_cef_execute_process_request{}))))
	if req == nil {
		return SubprocessExecutionResult{}, false, fmt.Errorf("allocate cef_execute_process request")
	}
	defer C.free(unsafe.Pointer(req))
	req.abi_revision = C.uint32_t(CurrentABIRevision)
	req.argc = C.uint64_t(len(cefArgs))
	req.argv = argv.ptr()

	out := (*C.eg_cef_subprocess_result)(C.calloc(1, C.size_t(unsafe.Sizeof(C.eg_cef_subprocess_result{}))))
	if out == nil {
		return SubprocessExecutionResult{}, false, fmt.Errorf("allocate cef_execute_process result")
	}
	defer C.free(unsafe.Pointer(out))

	status := C.eg_cef_shim_execute_process(req, out)
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
	cachePath := filepath.Join(absAppDir, ".electron-go", "cef-cache")
	if err := os.MkdirAll(cachePath, 0o755); err != nil {
		return CEFInitializeRequest{}, err
	}
	return NormalizeCEFInitializeRequest(CEFInitializeRequest{
		AppDir: absAppDir,
		Args:   appendCEFBrowserProcessSwitches(args),
		Settings: CEFSettings{
			NoSandbox:   true,
			CachePath:   cachePath,
			LogSeverity: CEFLogSeverityWarning,
		},
	}), nil
}

func appendCEFBrowserProcessSwitches(args []string) []string {
	out := append([]string(nil), args...)
	if IsCEFSubprocessArgs(out) {
		return out
	}
	seen := make(map[string]struct{}, len(out))
	for _, arg := range out {
		seen[arg] = struct{}{}
	}
	for _, flag := range cefBrowserProcessSwitches {
		if _, ok := seen[flag]; !ok {
			out = append(out, flag)
		}
	}
	return out
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

	cReq := (*C.eg_cef_initialize_request)(C.calloc(1, C.size_t(unsafe.Sizeof(C.eg_cef_initialize_request{}))))
	if cReq == nil {
		return fmt.Errorf("allocate cef_initialize request")
	}
	defer C.free(unsafe.Pointer(cReq))
	cReq.abi_revision = C.uint32_t(req.ABIRevision)
	cReq.app_dir = appDir.view
	cReq.argc = C.uint64_t(len(req.Args))
	cReq.argv = argv.ptr()
	cReq.settings.no_sandbox = boolToCUint8(req.Settings.NoSandbox)
	cReq.settings.cache_path = cachePath.view
	cReq.settings.log_severity = cLogSeverity(req.Settings.LogSeverity)

	status := C.eg_cef_shim_initialize(nil, cReq)
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

	urlView := newCStringView(req.URL)
	defer urlView.free()

	cReq := (*C.eg_browser_window_create_request)(C.calloc(1, C.size_t(unsafe.Sizeof(C.eg_browser_window_create_request{}))))
	if cReq == nil {
		return fmt.Errorf("allocate browser window create request")
	}
	defer C.free(unsafe.Pointer(cReq))
	cReq.abi_revision = C.uint32_t(req.ABIRevision)
	cReq.url = urlView.view
	cReq.width = C.int32_t(req.Width)
	cReq.height = C.int32_t(req.Height)
	cReq.show = boolToCUint8(req.Show)

	out := (*C.eg_browser_window_result)(C.calloc(1, C.size_t(unsafe.Sizeof(C.eg_browser_window_result{}))))
	if out == nil {
		return fmt.Errorf("allocate browser window create result")
	}
	defer C.free(unsafe.Pointer(out))

	status := C.eg_cef_shim_create_browser_sync(nil, cReq, out)
	if status != C.EG_BRIDGE_STATUS_RUNNING {
		return fmt.Errorf("CEF BrowserWindow create failed: status=%s", bridgeStatusName(status))
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
}

//export goOnBrowserLoadError
func goOnBrowserLoadError(browserID C.int, errorCode C.int) {
	cefLastBrowserID.Store(int64(browserID))
	cefLastLoadError.Store(int32(errorCode))
}

//export goOnBrowserBeforeClose
func goOnBrowserBeforeClose(browserID C.int) {
	cefLastBrowserID.Store(int64(browserID))
	cefBrowserClosed.Store(true)
}
