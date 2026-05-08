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
	ipcMain *IPCMain
	windows []*BrowserWindow
	modules map[string]goja.Value
	mu      sync.Mutex
	stdout  func(string)
}

type Option func(*Runtime)

func WithStdout(fn func(string)) Option {
	return func(r *Runtime) { r.stdout = fn }
}

func New(cwd string, opts ...Option) *Runtime {
	r := &Runtime{cwd: cwd, modules: map[string]goja.Value{}, stdout: func(s string) { fmt.Println(s) }}
	for _, opt := range opts {
		opt(r)
	}
	r.vm = goja.New()
	r.app = newApp(r)
	r.ipcMain = newIPCMain(r)
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
	if _, err := r.requireFrom(r.cwd, abs); err != nil {
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
		value, err := r.requireFrom(r.cwd, call.Arguments[0].String())
		if err != nil {
			panic(r.vm.ToValue(err.Error()))
		}
		return value
	})
}

func (r *Runtime) requireFrom(parentDir, name string) (goja.Value, error) {
	switch name {
	case "electron":
		return r.electronModule(), nil
	case "path":
		return r.pathModule(), nil
	}
	resolved, err := r.resolveModule(parentDir, name)
	if err != nil {
		return nil, err
	}
	if cached, ok := r.modules[resolved]; ok {
		return cached, nil
	}
	if filepath.Ext(resolved) == ".json" {
		data, err := os.ReadFile(resolved)
		if err != nil {
			return nil, err
		}
		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		value := r.vm.ToValue(v)
		r.modules[resolved] = value
		return value, nil
	}
	return r.runCommonJS(resolved)
}

func (r *Runtime) resolveModule(parentDir, name string) (string, error) {
	if isRelativeRequire(name) || filepath.IsAbs(name) {
		candidate := name
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(parentDir, name)
		}
		if resolved, ok := r.resolveAsFileOrDirectory(candidate); ok {
			return resolved, nil
		}
		return "", fmt.Errorf("golectron: cannot find module %q from %s", name, parentDir)
	}

	if resolved, ok := r.resolveNodeModule(parentDir, name); ok {
		return resolved, nil
	}
	return "", fmt.Errorf("golectron: cannot find module %q from %s", name, parentDir)
}

func (r *Runtime) resolveNodeModule(parentDir, name string) (string, bool) {
	for dir := parentDir; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "node_modules", name)
		if resolved, ok := r.resolveAsFileOrDirectory(candidate); ok {
			return resolved, true
		}
		if dir == filepath.Dir(dir) {
			break
		}
	}
	return "", false
}

func (r *Runtime) resolveAsFileOrDirectory(candidate string) (string, bool) {
	if resolved, ok := resolveAsFile(candidate); ok {
		return resolved, true
	}
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		if main, err := MainFileFromPackage(candidate); err == nil {
			if resolved, ok := r.resolveAsFileOrDirectory(filepath.Join(candidate, main)); ok {
				return resolved, true
			}
		}
		if resolved, ok := resolveAsFile(filepath.Join(candidate, "index")); ok {
			return resolved, true
		}
	}
	return "", false
}

func resolveAsFile(candidate string) (string, bool) {
	candidates := []string{candidate}
	if filepath.Ext(candidate) == "" {
		candidates = append(candidates, candidate+".js", candidate+".json")
	}
	for _, path := range candidates {
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			resolved, err := filepath.Abs(path)
			return resolved, err == nil
		}
	}
	return "", false
}

func (r *Runtime) runCommonJS(path string) (goja.Value, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	exports := r.vm.NewObject()
	module := r.vm.NewObject()
	_ = module.Set("exports", exports)
	r.modules[path] = exports
	wrapperSource := "(function(exports, module, require, __filename, __dirname) {\n" + string(data) + "\n})"
	wrapperValue, err := r.vm.RunScript(path, wrapperSource)
	if err != nil {
		return nil, err
	}
	wrapper, ok := goja.AssertFunction(wrapperValue)
	if !ok {
		return nil, fmt.Errorf("golectron: failed to compile module %s", path)
	}
	dir := filepath.Dir(path)
	require := func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(r.vm.ToValue("require() needs a module name"))
		}
		value, err := r.requireFrom(dir, call.Arguments[0].String())
		if err != nil {
			panic(r.vm.ToValue(err.Error()))
		}
		return value
	}
	if _, err := wrapper(goja.Undefined(), exports, module, r.vm.ToValue(require), r.vm.ToValue(path), r.vm.ToValue(dir)); err != nil {
		return nil, err
	}
	moduleExports := module.Get("exports")
	r.modules[path] = moduleExports
	return moduleExports, nil
}

func isRelativeRequire(name string) bool {
	return name == "." || name == ".." || len(name) >= 2 && (name[:2] == "./" || name[:2] == "..")
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
	_ = mod.Set("ipcMain", r.ipcMain.object())
	_ = mod.Set("ipcRenderer", r.ipcMain.rendererObject())
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

type IPCMain struct {
	*EventEmitter
	handlers map[string]goja.Callable
}

func newIPCMain(r *Runtime) *IPCMain {
	return &IPCMain{EventEmitter: newEventEmitter(r), handlers: map[string]goja.Callable{}}
}

func (i *IPCMain) object() *goja.Object {
	obj := i.EventEmitter.object()
	_ = obj.Set("handle", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(i.r.vm.ToValue("ipcMain.handle(channel, listener) requires a listener"))
		}
		fn, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			panic(i.r.vm.ToValue("ipcMain.handle listener must be a function"))
		}
		i.handlers[call.Argument(0).String()] = fn
		return goja.Undefined()
	})
	_ = obj.Set("removeHandler", func(channel string) { delete(i.handlers, channel) })
	return obj
}

func (i *IPCMain) rendererObject() *goja.Object {
	obj := i.EventEmitter.object()
	_ = obj.Set("invoke", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(i.r.vm.ToValue("ipcRenderer.invoke(channel, ...args) needs a channel"))
		}
		channel := call.Argument(0).String()
		handler, ok := i.handlers[channel]
		if !ok {
			panic(i.r.vm.ToValue("golectron: no ipcMain handler registered for " + channel))
		}
		event := i.r.vm.NewObject()
		_ = event.Set("sender", goja.Null())
		args := append([]goja.Value{event}, call.Arguments[1:]...)
		value, err := handler(goja.Undefined(), args...)
		if err != nil {
			panic(err)
		}
		return i.r.promiseResolved(value)
	})
	_ = obj.Set("send", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(i.r.vm.ToValue("ipcRenderer.send(channel, ...args) needs a channel"))
		}
		args := call.Arguments[1:]
		if err := i.emit(call.Argument(0).String(), args...); err != nil {
			panic(err)
		}
		return goja.Undefined()
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
	Preload string    `json:"preload,omitempty"`
	Shown   bool      `json:"shown"`
	Created time.Time `json:"created"`
	events  *EventEmitter
}

func newBrowserWindow(r *Runtime, opts goja.Value) *BrowserWindow {
	return &BrowserWindow{r: r, ID: len(r.windows) + 1, Options: opts.Export(), Preload: preloadPathFromOptions(opts.Export()), Created: time.Now(), events: newEventEmitter(r)}
}

func (w *BrowserWindow) object() *goja.Object {
	obj := w.events.object()
	_ = obj.Set("id", w.ID)
	_ = obj.Set("loadURL", func(url string) goja.Value {
		w.URL = url
		if err := w.runPreload(); err != nil {
			panic(w.r.vm.ToValue(err.Error()))
		}
		return w.r.promiseResolved(goja.Undefined())
	})
	_ = obj.Set("loadFile", func(path string) goja.Value {
		abs := path
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(w.r.cwd, path)
		}
		w.URL = "file://" + filepath.ToSlash(abs)
		if err := w.runPreload(); err != nil {
			panic(w.r.vm.ToValue(err.Error()))
		}
		return w.r.promiseResolved(goja.Undefined())
	})
	_ = obj.Set("show", func() { w.Shown = true })
	_ = obj.Set("hide", func() { w.Shown = false })
	_ = obj.Set("webContents", map[string]any{
		"send": func(channel string, args ...any) { _ = w.events.emit(channel) },
	})
	return obj
}

func (w *BrowserWindow) runPreload() error {
	if w.Preload == "" {
		return nil
	}
	_, err := w.r.requireFrom(w.r.cwd, w.Preload)
	return err
}

func preloadPathFromOptions(opts any) string {
	root, ok := opts.(map[string]any)
	if !ok {
		return ""
	}
	webPreferences, ok := root["webPreferences"].(map[string]any)
	if !ok {
		return ""
	}
	preload, _ := webPreferences["preload"].(string)
	return preload
}

const Version = "0.1.0-compat-spike"
