package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"

	"github.com/gibavargas/electron-go/internal/compat"
	"github.com/gibavargas/electron-go/internal/native"
	egruntime "github.com/gibavargas/electron-go/internal/runtime"
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
	checkParity := fs.Bool("check-parity", false, "exit successfully only when every ledger item is compatible")
	cefInitCheck := fs.Bool("cef-init-check", false, "initialize and shut down CEF without opening a window")
	runHello := fs.Bool("hello", false, "run the bundled hello conformance fixture")

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

	if *checkParity {
		if ledger.IsComplete() {
			fmt.Fprintf(os.Stdout, "electron-go parity complete for Electron %s\n", ledger.Target.Electron)
			return 0
		}
		fmt.Fprintf(os.Stderr, "electron-go parity incomplete for Electron %s: %d/%d compatible\n", ledger.Target.Electron, ledger.CompatibleCount(), ledger.TotalCount())
		return 1
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
