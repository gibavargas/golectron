package native

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
)

var ErrBridgeUnavailable = errors.New("native Chromium/Node/V8 bridge unavailable")

const CurrentABIRevision uint32 = 1

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

type StartRequest struct {
	ABIRevision     uint32
	AppDir          string
	MainPath        string
	AppName         string
	AppVersion      string
	ElectronVersion string
	Args            []string
	Environment     map[string]string
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

func NormalizeStartRequest(req StartRequest) StartRequest {
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
