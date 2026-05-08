package compat_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHelloFixtureConformance(t *testing.T) {
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

type fixtureResult struct {
	ExitCode int
	Output   string
}

func runFixture(t *testing.T, command, fixture string) fixtureResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	parts := strings.Fields(command)
	if len(parts) == 0 {
		t.Fatalf("empty command")
	}

	cmd := exec.CommandContext(ctx, parts[0], append(parts[1:], fixture)...)
	cmd.Env = os.Environ()
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
