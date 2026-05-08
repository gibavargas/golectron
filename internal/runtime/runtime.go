package runtime

import (
	"context"
	"fmt"
	"io"
	"os"

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
	Out             io.Writer
}

type Runtime struct {
	appDir          string
	electronVersion string
	bridge          native.Bridge
	out             io.Writer
	state           State
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
		out:             out,
		state:           StateCreated,
	}
}

func (r *Runtime) State() State {
	return r.state
}

func (r *Runtime) Run(ctx context.Context) error {
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
	r.state = StateStopped
	return nil
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
