package native

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestValidateStartRequest(t *testing.T) {
	valid := StartRequest{
		AppDir:          "/tmp/app",
		MainPath:        "/tmp/app/main.js",
		AppName:         "fixture",
		AppVersion:      "1.0.0",
		ElectronVersion: "42.0.0",
	}

	tests := []struct {
		name string
		req  StartRequest
		want string
	}{
		{name: "valid", req: valid},
		{name: "app dir", req: with(valid, func(req *StartRequest) { req.AppDir = " " }), want: "app directory is required"},
		{name: "main path", req: with(valid, func(req *StartRequest) { req.MainPath = "" }), want: "main path is required"},
		{name: "app name", req: with(valid, func(req *StartRequest) { req.AppName = "" }), want: "app name is required"},
		{name: "app version", req: with(valid, func(req *StartRequest) { req.AppVersion = "" }), want: "app version is required"},
		{name: "electron version", req: with(valid, func(req *StartRequest) { req.ElectronVersion = "" }), want: "electron version is required"},
		{name: "abi revision", req: with(valid, func(req *StartRequest) { req.ABIRevision = CurrentABIRevision + 1 }), want: "unsupported native bridge ABI revision"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStartRequest(tt.req)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("ValidateStartRequest() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateStartRequest() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestNormalizeStartRequestSetsCurrentABI(t *testing.T) {
	req := NormalizeStartRequest(StartRequest{})
	if req.ABIRevision != CurrentABIRevision {
		t.Fatalf("ABIRevision = %d, want %d", req.ABIRevision, CurrentABIRevision)
	}
}

func TestValidateCEFInitializeRequest(t *testing.T) {
	valid := CEFInitializeRequest{
		AppDir: "/tmp/app",
		Settings: CEFSettings{
			NoSandbox:   true,
			CachePath:   "/tmp/app/cache",
			LogSeverity: CEFLogSeverityWarning,
		},
	}

	tests := []struct {
		name string
		req  CEFInitializeRequest
		want string
	}{
		{name: "valid", req: valid},
		{name: "app dir", req: withCEFInitialize(valid, func(req *CEFInitializeRequest) { req.AppDir = " " }), want: "app directory is required"},
		{name: "cache path", req: withCEFInitialize(valid, func(req *CEFInitializeRequest) { req.Settings.CachePath = "" }), want: "CEF cache path is required"},
		{name: "log severity", req: withCEFInitialize(valid, func(req *CEFInitializeRequest) { req.Settings.LogSeverity = "trace" }), want: "unsupported CEF log severity"},
		{name: "abi revision", req: withCEFInitialize(valid, func(req *CEFInitializeRequest) { req.ABIRevision = CurrentABIRevision + 1 }), want: "unsupported native bridge ABI revision"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCEFInitializeRequest(tt.req)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("ValidateCEFInitializeRequest() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateCEFInitializeRequest() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestValidateBrowserWindowCreateRequest(t *testing.T) {
	valid := BrowserWindowCreateRequest{
		URL:    "https://example.test/",
		Width:  1024,
		Height: 768,
		Show:   true,
	}

	tests := []struct {
		name string
		req  BrowserWindowCreateRequest
		want string
	}{
		{name: "valid", req: valid},
		{name: "url", req: withBrowserWindowCreate(valid, func(req *BrowserWindowCreateRequest) { req.URL = "" }), want: "browser window URL is required"},
		{name: "width", req: withBrowserWindowCreate(valid, func(req *BrowserWindowCreateRequest) { req.Width = 0 }), want: "browser window width must be positive"},
		{name: "height", req: withBrowserWindowCreate(valid, func(req *BrowserWindowCreateRequest) { req.Height = -1 }), want: "browser window height must be positive"},
		{name: "abi revision", req: withBrowserWindowCreate(valid, func(req *BrowserWindowCreateRequest) { req.ABIRevision = CurrentABIRevision + 1 }), want: "unsupported native bridge ABI revision"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBrowserWindowCreateRequest(tt.req)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("ValidateBrowserWindowCreateRequest() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateBrowserWindowCreateRequest() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestValidateBrowserWindowLoadRequest(t *testing.T) {
	valid := BrowserWindowLoadRequest{
		BrowserID: 1,
		URL:       "https://example.test/",
	}

	tests := []struct {
		name string
		req  BrowserWindowLoadRequest
		want string
	}{
		{name: "valid", req: valid},
		{name: "browser id", req: withBrowserWindowLoad(valid, func(req *BrowserWindowLoadRequest) { req.BrowserID = 0 }), want: "browser ID must be positive"},
		{name: "url", req: withBrowserWindowLoad(valid, func(req *BrowserWindowLoadRequest) { req.URL = " " }), want: "browser window URL is required"},
		{name: "abi revision", req: withBrowserWindowLoad(valid, func(req *BrowserWindowLoadRequest) { req.ABIRevision = CurrentABIRevision + 1 }), want: "unsupported native bridge ABI revision"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBrowserWindowLoadRequest(tt.req)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("ValidateBrowserWindowLoadRequest() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateBrowserWindowLoadRequest() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestValidateBrowserWindowCloseRequest(t *testing.T) {
	valid := BrowserWindowCloseRequest{
		BrowserID: 1,
	}

	tests := []struct {
		name string
		req  BrowserWindowCloseRequest
		want string
	}{
		{name: "valid", req: valid},
		{name: "browser id", req: withBrowserWindowClose(valid, func(req *BrowserWindowCloseRequest) { req.BrowserID = 0 }), want: "browser ID must be positive"},
		{name: "abi revision", req: withBrowserWindowClose(valid, func(req *BrowserWindowCloseRequest) { req.ABIRevision = CurrentABIRevision + 1 }), want: "unsupported native bridge ABI revision"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBrowserWindowCloseRequest(tt.req)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("ValidateBrowserWindowCloseRequest() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateBrowserWindowCloseRequest() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestIsCEFSubprocessArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "renderer equals", args: []string{"electron-go", "--type=renderer"}, want: true},
		{name: "gpu split", args: []string{"electron-go", "--type", "gpu-process"}, want: true},
		{name: "browser process", args: []string{"electron-go", "./app"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCEFSubprocessArgs(tt.args); got != tt.want {
				t.Fatalf("IsCEFSubprocessArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStubBridgeValidatesBeforeUnavailable(t *testing.T) {
	_, err := (StubBridge{}).Start(context.Background(), StartRequest{})
	if err == nil || errors.Is(err, ErrBridgeUnavailable) {
		t.Fatalf("Start() error = %v, want validation error before unavailable", err)
	}
}

func TestStubBridgeReportsUnavailable(t *testing.T) {
	req := StartRequest{
		AppDir:          "/tmp/app",
		MainPath:        "/tmp/app/main.js",
		AppName:         "fixture",
		AppVersion:      "1.0.0",
		ElectronVersion: "42.0.0",
	}

	_, err := (StubBridge{}).Start(context.Background(), req)
	if !errors.Is(err, ErrBridgeUnavailable) {
		t.Fatalf("Start() error = %v, want ErrBridgeUnavailable", err)
	}
}

func TestUnavailableResult(t *testing.T) {
	result := UnavailableResult("testos")
	if result.Status != StatusUnavailable {
		t.Fatalf("Status = %s, want %s", result.Status, StatusUnavailable)
	}
	if result.Platform != "testos" {
		t.Fatalf("Platform = %q, want testos", result.Platform)
	}
	if result.BridgeRevision == "" {
		t.Fatal("BridgeRevision is empty")
	}
}

func TestIsAppActiveReportsPlatformSupport(t *testing.T) {
	activity, err := IsAppActive(context.Background())
	if err != nil {
		t.Fatalf("IsAppActive() error = %v", err)
	}
	if activity.Platform == "" {
		t.Fatal("Platform is empty")
	}
	if activity.Platform == "darwin" && !activity.Supported {
		t.Fatal("darwin AppKit app.isActive support is disabled")
	}
	if activity.Platform != "darwin" && activity.Supported {
		t.Fatalf("Supported = true on %s, want false", activity.Platform)
	}
	if !activity.Supported && activity.Active {
		t.Fatal("unsupported app.isActive report cannot be active")
	}
}

func with(req StartRequest, edit func(*StartRequest)) StartRequest {
	edit(&req)
	return req
}

func withCEFInitialize(req CEFInitializeRequest, edit func(*CEFInitializeRequest)) CEFInitializeRequest {
	edit(&req)
	return req
}

func withBrowserWindowCreate(req BrowserWindowCreateRequest, edit func(*BrowserWindowCreateRequest)) BrowserWindowCreateRequest {
	edit(&req)
	return req
}

func withBrowserWindowLoad(req BrowserWindowLoadRequest, edit func(*BrowserWindowLoadRequest)) BrowserWindowLoadRequest {
	edit(&req)
	return req
}

func withBrowserWindowClose(req BrowserWindowCloseRequest, edit func(*BrowserWindowCloseRequest)) BrowserWindowCloseRequest {
	edit(&req)
	return req
}
