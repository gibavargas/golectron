package electron

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dop251/goja"
)

// Runtime is a small Electron-compatible main-process runtime.
// It is intentionally strict and observable: unsupported APIs fail loudly instead
// of pretending to work.
type Runtime struct {
	vm      *goja.Runtime
	cwd     string
	app     *App
	windows []*BrowserWindow
	mu      sync.Mutex
	stdout  func(string)
}

type Option func(*Runtime)

func WithStdout(fn func(string)) Option {
	return func(r *Runtime) { r.stdout = fn }
}

func New(cwd string, opts ...Option) *Runtime {
	r := &Runtime{cwd: cwd, stdout: func(s string) { fmt.Println(s) }}
	for _, opt := range opts {
		opt(r)
	}
	r.vm = goja.New()
	r.app = newApp(r)
	r.installGlobals()
	return r
}

func (r *Runtime) Windows() []*BrowserWindow {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*BrowserWindow, len(r.windows))
	copy(out, r.windows)
	return out
}

func (r *Runtime) RunFile(path string) error {
	abs := path
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(r.cwd, path)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return err
	}
	_, err = r.vm.RunScript(abs, string(data))
	if err != nil {
		return err
	}
	return r.app.emit("ready")
}

func (r *Runtime) installGlobals() {
	console := r.vm.NewObject()
	_ = console.Set("log", func(call goja.FunctionCall) goja.Value {
		parts := make([]any, 0, len(call.Arguments))
		for _, arg := range call.Arguments {
			parts = append(parts, arg.Export())
		}
		r.stdout(fmt.Sprint(parts...))
		return goja.Undefined()
	})
	_ = r.vm.Set("console", console)

	_ = r.vm.Set("require", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(r.vm.ToValue("require() needs a module name"))
		}
		name := call.Arguments[0].String()
		switch name {
		case "electron":
			return r.electronModule()
		case "path":
			return r.pathModule()
		default:
			panic(r.vm.ToValue("golectron: unsupported require(" + name + ")"))
		}
	})
}

func (r *Runtime) electronModule() goja.Value {
	mod := r.vm.NewObject()
	_ = mod.Set("app", r.app.object())
	_ = mod.Set("BrowserWindow", func(call goja.ConstructorCall) *goja.Object {
		bw := newBrowserWindow(r, call.Argument(0))
		r.mu.Lock()
		r.windows = append(r.windows, bw)
		r.mu.Unlock()
		return bw.object()
	})
	_ = mod.Set("ipcMain", newEventEmitter(r).object())
	_ = mod.Set("dialog", map[string]any{
		"showErrorBox": func(title, content string) { r.stdout("ERROR: " + title + ": " + content) },
	})
	return mod
}

func (r *Runtime) pathModule() goja.Value {
	mod := r.vm.NewObject()
	_ = mod.Set("join", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, 0, len(call.Arguments))
		for _, arg := range call.Arguments {
			parts = append(parts, arg.String())
		}
		return r.vm.ToValue(filepath.Join(parts...))
	})
	return mod
}

func (r *Runtime) promiseResolved(v goja.Value) goja.Value {
	p, resolve, _ := r.vm.NewPromise()
	resolve(v)
	return r.vm.ToValue(p)
}

type PackageJSON struct {
	Main string `json:"main"`
}

func MainFileFromPackage(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return "", err
	}
	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", err
	}
	if pkg.Main == "" {
		return "", errors.New("package.json has no main field")
	}
	return pkg.Main, nil
}

type EventEmitter struct {
	r         *Runtime
	listeners map[string][]goja.Callable
}

func newEventEmitter(r *Runtime) *EventEmitter {
	return &EventEmitter{r: r, listeners: map[string][]goja.Callable{}}
}

func (e *EventEmitter) on(name string, fn goja.Callable) {
	e.listeners[name] = append(e.listeners[name], fn)
}

func (e *EventEmitter) emit(name string, args ...goja.Value) error {
	for _, fn := range e.listeners[name] {
		if _, err := fn(goja.Undefined(), args...); err != nil {
			return err
		}
	}
	return nil
}

func (e *EventEmitter) object() *goja.Object {
	obj := e.r.vm.NewObject()
	_ = obj.Set("on", func(call goja.FunctionCall) goja.Value {
		fn, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			panic(e.r.vm.ToValue("listener must be a function"))
		}
		e.on(call.Argument(0).String(), fn)
		return obj
	})
	_ = obj.Set("once", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		fn, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			panic(e.r.vm.ToValue("listener must be a function"))
		}
		var wrapper goja.Callable
		wrapper = func(this goja.Value, args ...goja.Value) (goja.Value, error) {
			e.listeners[name] = nil
			return fn(this, args...)
		}
		e.on(name, wrapper)
		return obj
	})
	_ = obj.Set("emit", func(call goja.FunctionCall) goja.Value {
		args := call.Arguments[1:]
		if err := e.emit(call.Argument(0).String(), args...); err != nil {
			panic(err)
		}
		return e.r.vm.ToValue(true)
	})
	return obj
}

type App struct {
	*EventEmitter
	ready bool
}

func newApp(r *Runtime) *App { return &App{EventEmitter: newEventEmitter(r)} }

func (a *App) object() *goja.Object {
	obj := a.EventEmitter.object()
	_ = obj.Set("whenReady", func() goja.Value { return a.r.promiseResolved(a.r.vm.ToValue(true)) })
	_ = obj.Set("isReady", func() bool { return a.ready })
	_ = obj.Set("quit", func() { _ = a.emit("quit") })
	_ = obj.Set("getVersion", func() string { return Version })
	return obj
}

func (a *App) emit(name string, args ...goja.Value) error {
	if name == "ready" {
		a.ready = true
	}
	return a.EventEmitter.emit(name, args...)
}

type BrowserWindow struct {
	r       *Runtime
	ID      int       `json:"id"`
	Options any       `json:"options,omitempty"`
	URL     string    `json:"url,omitempty"`
	Shown   bool      `json:"shown"`
	Created time.Time `json:"created"`
	events  *EventEmitter
}

func newBrowserWindow(r *Runtime, opts goja.Value) *BrowserWindow {
	return &BrowserWindow{r: r, ID: len(r.windows) + 1, Options: opts.Export(), Created: time.Now(), events: newEventEmitter(r)}
}

func (w *BrowserWindow) object() *goja.Object {
	obj := w.events.object()
	_ = obj.Set("id", w.ID)
	_ = obj.Set("loadURL", func(url string) goja.Value {
		w.URL = url
		return w.r.promiseResolved(goja.Undefined())
	})
	_ = obj.Set("show", func() { w.Shown = true })
	_ = obj.Set("hide", func() { w.Shown = false })
	_ = obj.Set("webContents", map[string]any{
		"send": func(channel string, args ...any) { _ = w.events.emit(channel) },
	})
	return obj
}

const Version = "0.1.0-compat-spike"
