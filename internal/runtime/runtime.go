package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gibavargas/electron-go/internal/appmeta"
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

const StartupTraceEnv = "ELECTRON_GO_STARTUP_TRACE"

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
		Args:        append([]string(nil), args...),
		Environment: append([]string(nil), environment...),
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
	fmt.Fprintf(r.out, "electron-go: loaded %s %s\n", meta.Name, meta.Version)

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
	result, err := r.bridge.Start(ctx, req)
	if err != nil {
		return err
	}

	r.state = StateRunning
	fmt.Fprintf(r.out, "electron-go: running pid=%d windows=%d chromium=%s node=%s v8=%s\n", result.PID, result.WindowCount, result.Chromium, result.Node, result.V8)
	if envEnabled(r.environment, StartupTraceEnv) && len(result.StartupTraceMS) > 0 {
		payload, err := json.Marshal(result.StartupTraceMS)
		if err == nil {
			fmt.Fprintf(r.out, "electron-go-startup-trace: %s\n", payload)
		}
	}
	r.state = StateStopped
	return nil
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
