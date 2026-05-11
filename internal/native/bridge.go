package native

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
)

var ErrBridgeUnavailable = errors.New("native Chromium/Node/V8 bridge unavailable")
var ErrCEFBrowserWindowNotImplemented = errors.New("CEF BrowserWindow bootstrap not implemented")

const CurrentABIRevision uint32 = 1

const CEFSubprocessUnavailableExitCode = 78

type Status string

const (
	StatusUnavailable    Status = "unavailable"
	StatusInvalidRequest Status = "invalid-request"
	StatusStarting       Status = "starting"
	StatusRunning        Status = "running"
	StatusStopped        Status = "stopped"
)

type EngineVersions struct {
	Chromium string
	Node     string
	V8       string
}

type CEFLogSeverity string

const (
	CEFLogSeverityDefault CEFLogSeverity = ""
	CEFLogSeverityVerbose CEFLogSeverity = "verbose"
	CEFLogSeverityInfo    CEFLogSeverity = "info"
	CEFLogSeverityWarning CEFLogSeverity = "warning"
	CEFLogSeverityError   CEFLogSeverity = "error"
	CEFLogSeverityFatal   CEFLogSeverity = "fatal"
	CEFLogSeverityDisable CEFLogSeverity = "disable"
)

// CEFSettings mirrors the small bootstrap subset exposed by the native C ABI.
// The native bridge must be initialized and shut down on the process main
// thread; callbacks crossing this boundary must be short-lived, non-blocking,
// and must not retain Go pointers after returning.
type CEFSettings struct {
	NoSandbox   bool
	CachePath   string
	LogSeverity CEFLogSeverity
}

type CEFInitializeRequest struct {
	ABIRevision uint32
	AppDir      string
	Args        []string
	Settings    CEFSettings
}

type BrowserWindowCreateRequest struct {
	ABIRevision uint32
	URL         string
	Width       int
	Height      int
	Show        bool
	// AutoCloseOnLoad preserves the current bootstrap behavior where a CEF
	// BrowserWindow closes after the main frame loads. Runtime-driven windows
	// should leave this false and close from app/BrowserWindow lifecycle calls.
	AutoCloseOnLoad bool
}

type BrowserWindowLoadRequest struct {
	ABIRevision uint32
	BrowserID   int64
	URL         string
}

type BrowserWindowCloseRequest struct {
	ABIRevision uint32
	BrowserID   int64
}

type BrowserWindowScriptRequest struct {
	ABIRevision uint32
	BrowserID   int64
	Script      string
}

type SubprocessExecutionResult struct {
	ExitCode int
	Status   Status
	Error    string
}

type StartRequest struct {
	ABIRevision     uint32
	AppDir          string
	MainPath        string
	AppName         string
	AppVersion      string
	ElectronVersion string
	Args            []string
	Environment     []string
	NodeOptions     NativeNodeOptions
}

type NativeNodeOptions struct {
	ExperimentalTransformTypes bool
}

type StartResult struct {
	PID            int
	WindowCount    int
	Status         Status
	Chromium       string
	Node           string
	V8             string
	Compatibility  string
	BridgeRevision string
	Platform       string
	StartupTraceMS map[string]int64
}

type Bridge interface {
	Start(context.Context, StartRequest) (*StartResult, error)
}

type StubBridge struct {
	Platform string
}

func (b StubBridge) Start(ctx context.Context, req StartRequest) (*StartResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateStartRequest(req); err != nil {
		return nil, err
	}
	return nil, ErrBridgeUnavailable
}

func ValidateStartRequest(req StartRequest) error {
	if req.ABIRevision != 0 && req.ABIRevision != CurrentABIRevision {
		return fmt.Errorf("unsupported native bridge ABI revision: %d", req.ABIRevision)
	}
	if strings.TrimSpace(req.AppDir) == "" {
		return fmt.Errorf("app directory is required")
	}
	if strings.TrimSpace(req.MainPath) == "" {
		return fmt.Errorf("main path is required")
	}
	if strings.TrimSpace(req.AppName) == "" {
		return fmt.Errorf("app name is required")
	}
	if strings.TrimSpace(req.AppVersion) == "" {
		return fmt.Errorf("app version is required")
	}
	if strings.TrimSpace(req.ElectronVersion) == "" {
		return fmt.Errorf("electron version is required")
	}
	return nil
}

func IsCEFSubprocessArgs(args []string) bool {
	for _, arg := range args {
		if arg == "--type" || strings.HasPrefix(arg, "--type=") {
			return true
		}
	}
	return false
}

func ValidateCEFInitializeRequest(req CEFInitializeRequest) error {
	if err := validateABIRevision(req.ABIRevision); err != nil {
		return err
	}
	if strings.TrimSpace(req.AppDir) == "" {
		return fmt.Errorf("app directory is required")
	}
	if strings.TrimSpace(req.Settings.CachePath) == "" {
		return fmt.Errorf("CEF cache path is required")
	}
	return validateCEFLogSeverity(req.Settings.LogSeverity)
}

func ValidateBrowserWindowCreateRequest(req BrowserWindowCreateRequest) error {
	if err := validateABIRevision(req.ABIRevision); err != nil {
		return err
	}
	if strings.TrimSpace(req.URL) == "" {
		return fmt.Errorf("browser window URL is required")
	}
	if req.Width <= 0 {
		return fmt.Errorf("browser window width must be positive")
	}
	if req.Height <= 0 {
		return fmt.Errorf("browser window height must be positive")
	}
	return nil
}

func ValidateBrowserWindowLoadRequest(req BrowserWindowLoadRequest) error {
	if err := validateABIRevision(req.ABIRevision); err != nil {
		return err
	}
	if req.BrowserID <= 0 {
		return fmt.Errorf("browser ID must be positive")
	}
	if strings.TrimSpace(req.URL) == "" {
		return fmt.Errorf("browser window URL is required")
	}
	return nil
}

func ValidateBrowserWindowCloseRequest(req BrowserWindowCloseRequest) error {
	if err := validateABIRevision(req.ABIRevision); err != nil {
		return err
	}
	if req.BrowserID <= 0 {
		return fmt.Errorf("browser ID must be positive")
	}
	return nil
}

func ValidateBrowserWindowScriptRequest(req BrowserWindowScriptRequest) error {
	if err := validateABIRevision(req.ABIRevision); err != nil {
		return err
	}
	if req.BrowserID <= 0 {
		return fmt.Errorf("browser ID must be positive")
	}
	if strings.TrimSpace(req.Script) == "" {
		return fmt.Errorf("browser window script is required")
	}
	return nil
}

func NormalizeStartRequest(req StartRequest) StartRequest {
	if req.ABIRevision == 0 {
		req.ABIRevision = CurrentABIRevision
	}
	return req
}

func NormalizeCEFInitializeRequest(req CEFInitializeRequest) CEFInitializeRequest {
	if req.ABIRevision == 0 {
		req.ABIRevision = CurrentABIRevision
	}
	return req
}

func NormalizeBrowserWindowCreateRequest(req BrowserWindowCreateRequest) BrowserWindowCreateRequest {
	if req.ABIRevision == 0 {
		req.ABIRevision = CurrentABIRevision
	}
	return req
}

func NormalizeBrowserWindowLoadRequest(req BrowserWindowLoadRequest) BrowserWindowLoadRequest {
	if req.ABIRevision == 0 {
		req.ABIRevision = CurrentABIRevision
	}
	return req
}

func NormalizeBrowserWindowCloseRequest(req BrowserWindowCloseRequest) BrowserWindowCloseRequest {
	if req.ABIRevision == 0 {
		req.ABIRevision = CurrentABIRevision
	}
	return req
}

func NormalizeBrowserWindowScriptRequest(req BrowserWindowScriptRequest) BrowserWindowScriptRequest {
	if req.ABIRevision == 0 {
		req.ABIRevision = CurrentABIRevision
	}
	return req
}

func UnavailableResult(platform string) StartResult {
	if platform == "" {
		platform = runtime.GOOS
	}
	return StartResult{
		Status:         StatusUnavailable,
		BridgeRevision: fmt.Sprintf("abi-%d-stub", CurrentABIRevision),
		Platform:       platform,
	}
}

func validateABIRevision(revision uint32) error {
	if revision != 0 && revision != CurrentABIRevision {
		return fmt.Errorf("unsupported native bridge ABI revision: %d", revision)
	}
	return nil
}

func validateCEFLogSeverity(severity CEFLogSeverity) error {
	switch severity {
	case CEFLogSeverityDefault,
		CEFLogSeverityVerbose,
		CEFLogSeverityInfo,
		CEFLogSeverityWarning,
		CEFLogSeverityError,
		CEFLogSeverityFatal,
		CEFLogSeverityDisable:
		return nil
	default:
		return fmt.Errorf("unsupported CEF log severity: %q", severity)
	}
}
