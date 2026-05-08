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

func with(req StartRequest, edit func(*StartRequest)) StartRequest {
	edit(&req)
	return req
}
