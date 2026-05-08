package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jvguidi/golectron/pkg/electron"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "golectron:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return runApp(".")
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Println(electron.Version)
		return nil
	case "init":
		return initExample(".")
	case "run":
		fs := flag.NewFlagSet("run", flag.ContinueOnError)
		dir := fs.String("dir", ".", "Electron app directory")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return runApp(*dir)
	default:
		return runApp(args[0])
	}
}

func runApp(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	mainFile, err := electron.MainFileFromPackage(abs)
	if err != nil {
		return err
	}
	r := electron.New(abs)
	if err := r.RunFile(mainFile); err != nil {
		return err
	}
	for _, w := range r.Windows() {
		fmt.Printf("window[%d] url=%s shown=%v\n", w.ID, w.URL, w.Shown)
	}
	return nil
}

func initExample(dir string) error {
	pkg := `{"main":"main.js","scripts":{"start":"golectron run"}}` + "\n"
	main := `const { app, BrowserWindow } = require('electron')

app.whenReady().then(() => {
  const win = new BrowserWindow({ width: 900, height: 600 })
  win.loadURL('https://example.com')
  win.show()
})
` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "main.js"), []byte(main), 0644)
}
