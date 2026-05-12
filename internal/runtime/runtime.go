package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	goruntime "runtime"
	"slices"
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
	AppDir            string
	ElectronVersion   string
	Bridge            native.Bridge
	SubprocessHook    SubprocessHook
	Args              []string
	Environment       []string
	NodeOptions       NodeOptions
	Out               io.Writer
	SkipFinalShutdown bool
}

type Runtime struct {
	appDir            string
	electronVersion   string
	bridge            native.Bridge
	subprocessHook    SubprocessHook
	args              []string
	environment       []string
	nodeOptions       NodeOptions
	out               io.Writer
	skipFinalShutdown bool
	state             State
}

type WarmRunReport struct {
	Iterations   int          `json:"iterations"`
	InitializeMS int64        `json:"initialize_ms"`
	ShutdownMS   int64        `json:"shutdown_ms"`
	Samples      []WarmSample `json:"samples"`
	Summary      WarmSummary  `json:"summary"`
}

type WarmSample struct {
	Iteration      int              `json:"iteration"`
	DurationMS     int64            `json:"duration_ms"`
	StartupTraceMS map[string]int64 `json:"startup_trace_ms,omitempty"`
	ExitCode       int              `json:"exit_code"`
}

type WarmSummary struct {
	Successes        int     `json:"successes"`
	Failures         int     `json:"failures"`
	DurationMinMS    int64   `json:"duration_min_ms"`
	DurationMedianMS int64   `json:"duration_median_ms"`
	DurationMeanMS   float64 `json:"duration_mean_ms"`
	DurationMaxMS    int64   `json:"duration_max_ms"`
}

type mainPlanBridge interface {
	mainrunner.Driver
	InitializeForStart(context.Context, native.StartRequest) error
	Shutdown(context.Context) error
}

type validatedMainPlanBridge interface {
	InitializeValidatedForStart(context.Context, native.StartRequest) error
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
		appDir:            opts.AppDir,
		electronVersion:   opts.ElectronVersion,
		bridge:            bridge,
		subprocessHook:    opts.SubprocessHook,
		args:              append([]string(nil), opts.Args...),
		environment:       append([]string(nil), opts.Environment...),
		nodeOptions:       opts.NodeOptions,
		out:               out,
		skipFinalShutdown: opts.SkipFinalShutdown,
		state:             StateCreated,
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
		Args:            r.args,
		Environment:     r.environment,
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

func (r *Runtime) WarmRun(ctx context.Context, iterations int) (WarmRunReport, error) {
	if iterations <= 0 {
		return WarmRunReport{}, fmt.Errorf("warm run iterations must be positive")
	}
	if iterations > 1 {
		return WarmRunReport{}, fmt.Errorf("warm run currently supports one visible window per CEF initialization")
	}
	bridge, ok := r.bridge.(mainPlanBridge)
	if !ok {
		return WarmRunReport{}, fmt.Errorf("warm run requires native main-plan bridge")
	}
	meta, err := appmeta.Load(r.appDir)
	if err != nil {
		return WarmRunReport{}, err
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
		return WarmRunReport{}, err
	}
	plan, err := mainrunner.ParseFile(req.MainPath)
	if err != nil {
		return WarmRunReport{}, err
	}

	report := WarmRunReport{Iterations: iterations, Samples: make([]WarmSample, 0, iterations)}
	stageStart := time.Now()
	if err := bridge.InitializeForStart(ctx, req); err != nil {
		return report, err
	}
	report.InitializeMS = time.Since(stageStart).Milliseconds()
	var runErr error
	for i := 1; i <= iterations; i++ {
		sampleStart := time.Now()
		execResult, err := mainrunner.Execute(ctx, plan, bridge, mainrunner.ExecuteOptions{
			Environment: r.environment,
			Out:         r.out,
			Trace:       true,
		})
		sample := WarmSample{
			Iteration:      i,
			DurationMS:     time.Since(sampleStart).Milliseconds(),
			StartupTraceMS: execResult.TraceMS,
			ExitCode:       execResult.ExitCode,
		}
		if err != nil {
			sample.ExitCode = 1
			runErr = err
		}
		report.Samples = append(report.Samples, sample)
		if err != nil {
			break
		}
	}
	report.Summary = summarizeWarmSamples(report.Samples)
	stageStart = time.Now()
	shutdownErr := bridge.Shutdown(ctx)
	report.ShutdownMS = time.Since(stageStart).Milliseconds()
	if runErr != nil {
		return report, runErr
	}
	if shutdownErr != nil {
		return report, shutdownErr
	}
	return report, nil
}

func summarizeWarmSamples(samples []WarmSample) WarmSummary {
	summary := WarmSummary{}
	durations := make([]int64, 0, len(samples))
	for _, sample := range samples {
		if sample.ExitCode == 0 {
			summary.Successes++
			durations = append(durations, sample.DurationMS)
			continue
		}
		summary.Failures++
	}
	if len(durations) == 0 {
		return summary
	}
	slices.Sort(durations)
	summary.DurationMinMS = durations[0]
	summary.DurationMedianMS = durations[len(durations)/2]
	summary.DurationMaxMS = durations[len(durations)-1]
	var total int64
	for _, duration := range durations {
		total += duration
	}
	summary.DurationMeanMS = float64(total) / float64(len(durations))
	return summary
}

func (r *Runtime) startBridge(ctx context.Context, req native.StartRequest) (*native.StartResult, error) {
	if bridge, ok := r.bridge.(mainPlanBridge); ok {
		return r.startMainPlanBridge(ctx, req, bridge)
	}
	return r.bridge.Start(ctx, req)
}

func (r *Runtime) startMainPlanBridge(ctx context.Context, req native.StartRequest, bridge mainPlanBridge) (*native.StartResult, error) {
	plan, err := mainrunner.ParseFile(req.MainPath)
	if err == nil {
		traceEnabled := envEnabled(r.environment, StartupTraceEnv)
		var traceStart time.Time
		var trace map[string]int64
		if traceEnabled {
			traceStart = time.Now()
		}
		var stageStart time.Time
		if traceEnabled {
			stageStart = time.Now()
		}
		if err := initializeMainPlanBridge(ctx, bridge, req); err != nil {
			return nil, err
		}
		if traceEnabled {
			trace = map[string]int64{
				"cef_initialize": time.Since(stageStart).Milliseconds(),
			}
			stageStart = time.Now()
		}
		var executeOffset int64
		if traceEnabled {
			executeOffset = time.Since(traceStart).Milliseconds()
		}
		execResult, execErr := mainrunner.Execute(ctx, plan, bridge, mainrunner.ExecuteOptions{
			Environment: r.environment,
			Out:         r.out,
			Trace:       traceEnabled,
		})
		if traceEnabled {
			trace["mainrunner_execute"] = time.Since(stageStart).Milliseconds()
			for key, value := range execResult.TraceMS {
				trace[key] = executeOffset + value
			}
		}
		var shutdownErr error
		if r.skipFinalShutdown && execErr == nil {
			if traceEnabled {
				trace["cef_shutdown_skipped"] = 1
			}
		} else {
			if traceEnabled {
				stageStart = time.Now()
			}
			shutdownErr = bridge.Shutdown(ctx)
			if traceEnabled {
				trace["cef_shutdown"] = time.Since(stageStart).Milliseconds()
			}
		}
		if traceEnabled {
			trace["total_native_start"] = time.Since(traceStart).Milliseconds()
		}
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

func initializeMainPlanBridge(ctx context.Context, bridge mainPlanBridge, req native.StartRequest) error {
	if validated, ok := bridge.(validatedMainPlanBridge); ok {
		return validated.InitializeValidatedForStart(ctx, req)
	}
	return bridge.InitializeForStart(ctx, req)
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
