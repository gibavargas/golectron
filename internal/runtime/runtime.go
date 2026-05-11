package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/gibavargas/electron-go/internal/appmeta"
	"github.com/gibavargas/electron-go/internal/mainrunner"
	"github.com/gibavargas/electron-go/internal/native"
)

type State string

const (
	StateCreated       State = "created"
	StateMetadataReady State = "metadata-ready"
	StateStarting      State = "starting"
	StateRunning       State = "running"
	StateStopped       State = "stopped"
)

type Options struct {
	AppDir          string
	ElectronVersion string
	Bridge          native.Bridge
	SubprocessHook  SubprocessHook
	Args            []string
	Environment     []string
	NodeOptions     NodeOptions
	Out             io.Writer
}

type Runtime struct {
	appDir          string
	electronVersion string
	bridge          native.Bridge
	subprocessHook  SubprocessHook
	args            []string
	environment     []string
	nodeOptions     NodeOptions
	out             io.Writer
	state           State
}

type mainPlanBridge interface {
	mainrunner.Driver
	InitializeForStart(context.Context, native.StartRequest) error
	Shutdown(context.Context) error
}

const StartupTraceEnv = "ELECTRON_GO_STARTUP_TRACE"
const VerboseEnv = "ELECTRON_GO_VERBOSE"
const ScopedMainRunnerEnv = "ELECTRON_GO_ENABLE_SCOPED_MAIN_RUNNER"

type SubprocessRequest struct {
	Args        []string
	Environment []string
}

type SubprocessResult struct {
	Handled  bool
	ExitCode int
}

type SubprocessHook func(context.Context, SubprocessRequest) (SubprocessResult, error)

func NativeSubprocessHook(ctx context.Context, req SubprocessRequest) (SubprocessResult, error) {
	if err := ctx.Err(); err != nil {
		return SubprocessResult{}, err
	}
	if !native.IsCEFSubprocessArgs(req.Args) {
		return SubprocessResult{}, nil
	}
	result, handled, err := native.ExecuteCEFSubprocess(ctx, req.Args)
	if err != nil {
		return SubprocessResult{}, err
	}
	if handled {
		return SubprocessResult{
			Handled:  true,
			ExitCode: result.ExitCode,
		}, nil
	}
	return SubprocessResult{}, nil
}

func ExecuteSubprocess(ctx context.Context, hook SubprocessHook, args []string, environment []string) (SubprocessResult, error) {
	if hook == nil {
		return SubprocessResult{}, nil
	}
	return hook(ctx, SubprocessRequest{
		Args:        args,
		Environment: environment,
	})
}

type SubprocessExit struct {
	Code int
}

func (e *SubprocessExit) Error() string {
	return fmt.Sprintf("native subprocess exited with code %d", e.Code)
}

func New(opts Options) *Runtime {
	out := opts.Out
	if out == nil {
		out = io.Discard
	}
	bridge := opts.Bridge
	if bridge == nil {
		bridge = native.StubBridge{}
	}
	return &Runtime{
		appDir:          opts.AppDir,
		electronVersion: opts.ElectronVersion,
		bridge:          bridge,
		subprocessHook:  opts.SubprocessHook,
		args:            append([]string(nil), opts.Args...),
		environment:     append([]string(nil), opts.Environment...),
		nodeOptions:     opts.NodeOptions,
		out:             out,
		state:           StateCreated,
	}
}

func (r *Runtime) State() State {
	return r.state
}

func (r *Runtime) Run(ctx context.Context) error {
	if r.subprocessHook != nil {
		result, err := ExecuteSubprocess(ctx, r.subprocessHook, r.args, r.environment)
		if err != nil {
			return err
		}
		if result.Handled {
			r.state = StateStopped
			return &SubprocessExit{Code: result.ExitCode}
		}
	}

	meta, err := appmeta.Load(r.appDir)
	if err != nil {
		return err
	}
	r.state = StateMetadataReady
	verbose := envEnabled(r.environment, VerboseEnv)
	if verbose {
		fmt.Fprintf(r.out, "electron-go: loaded %s %s\n", meta.Name, meta.Version)
	}

	req := native.NormalizeStartRequest(native.StartRequest{
		AppDir:          meta.Dir,
		MainPath:        meta.MainPath(),
		AppName:         meta.Name,
		AppVersion:      meta.Version,
		ElectronVersion: r.electronVersion,
		Args:            append([]string(nil), r.args...),
		Environment:     append([]string(nil), r.environment...),
		NodeOptions: native.NativeNodeOptions{
			ExperimentalTransformTypes: r.nodeOptions.ExperimentalTransformTypes,
		},
	})
	if err := native.ValidateStartRequest(req); err != nil {
		return err
	}

	r.state = StateStarting
	result, err := r.startBridge(ctx, req)
	if err != nil {
		return err
	}

	r.state = StateRunning
	if verbose {
		fmt.Fprintf(r.out, "electron-go: running pid=%d windows=%d chromium=%s node=%s v8=%s\n", result.PID, result.WindowCount, result.Chromium, result.Node, result.V8)
	}
	if envEnabled(r.environment, StartupTraceEnv) && len(result.StartupTraceMS) > 0 {
		payload, err := json.Marshal(result.StartupTraceMS)
		if err == nil {
			fmt.Fprintf(r.out, "electron-go-startup-trace: %s\n", payload)
		}
	}
	r.state = StateStopped
	return nil
}

func (r *Runtime) startBridge(ctx context.Context, req native.StartRequest) (*native.StartResult, error) {
	if envEnabled(r.environment, ScopedMainRunnerEnv) {
		if bridge, ok := r.bridge.(mainPlanBridge); ok {
			return r.startMainPlanBridge(ctx, req, bridge)
		}
	}
	return r.bridge.Start(ctx, req)
}

func (r *Runtime) startMainPlanBridge(ctx context.Context, req native.StartRequest, bridge mainPlanBridge) (*native.StartResult, error) {
	plan, err := mainrunner.ParseFile(req.MainPath)
	if err == nil {
		traceStart := time.Now()
		stageStart := traceStart
		if err := bridge.InitializeForStart(ctx, req); err != nil {
			return nil, err
		}
		trace := map[string]int64{
			"cef_initialize": time.Since(stageStart).Milliseconds(),
		}
		stageStart = time.Now()
		execResult, execErr := mainrunner.Execute(ctx, plan, bridge, mainrunner.ExecuteOptions{
			Environment: r.environment,
			Out:         r.out,
		})
		trace["mainrunner_execute"] = time.Since(stageStart).Milliseconds()
		for key, value := range execResult.TraceMS {
			trace[key] = value
		}
		stageStart = time.Now()
		shutdownErr := bridge.Shutdown(ctx)
		trace["cef_shutdown"] = time.Since(stageStart).Milliseconds()
		trace["total_native_start"] = time.Since(traceStart).Milliseconds()
		if execErr != nil {
			return nil, execErr
		}
		if shutdownErr != nil {
			return nil, shutdownErr
		}
		return &native.StartResult{
			PID:            os.Getpid(),
			WindowCount:    1,
			Status:         native.StatusStopped,
			BridgeRevision: fmt.Sprintf("abi-%d-mainrunner", native.CurrentABIRevision),
			Platform:       goruntime.GOOS,
			StartupTraceMS: trace,
		}, nil
	}
	if !errors.Is(err, mainrunner.ErrUnsupportedScript) {
		return nil, err
	}
	return r.bridge.Start(ctx, req)
}

func envEnabled(env []string, key string) bool {
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			value := strings.TrimSpace(strings.TrimPrefix(entry, prefix))
			return value != "" && value != "0" && !strings.EqualFold(value, "false")
		}
	}
	return false
}

func AppDirFromArgs(args []string) string {
	if len(args) == 0 || args[0] == "" {
		wd, err := os.Getwd()
		if err == nil {
			return wd
		}
		return "."
	}
	return args[0]
}
