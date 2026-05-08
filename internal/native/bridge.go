package native

import (
	"context"
	"errors"
	"fmt"
)

var ErrBridgeUnavailable = errors.New("native Chromium/Node/V8 bridge unavailable")

type StartRequest struct {
	AppDir          string
	MainPath        string
	AppName         string
	AppVersion      string
	ElectronVersion string
}

type StartResult struct {
	PID            int
	WindowCount    int
	Chromium       string
	Node           string
	V8             string
	Compatibility  string
	BridgeRevision string
}

type Bridge interface {
	Start(context.Context, StartRequest) (*StartResult, error)
}

type StubBridge struct{}

func (StubBridge) Start(context.Context, StartRequest) (*StartResult, error) {
	return nil, ErrBridgeUnavailable
}

func ValidateStartRequest(req StartRequest) error {
	if req.AppDir == "" {
		return fmt.Errorf("app directory is required")
	}
	if req.MainPath == "" {
		return fmt.Errorf("main path is required")
	}
	if req.AppName == "" {
		return fmt.Errorf("app name is required")
	}
	if req.ElectronVersion == "" {
		return fmt.Errorf("electron version is required")
	}
	return nil
}
