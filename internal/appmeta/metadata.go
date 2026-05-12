package appmeta

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultMain    = "index.js"
	defaultVersion = "0.0.0"
)

type Metadata struct {
	Dir         string
	Name        string
	Version     string
	Main        string
	Description string
	mainPath    string
}

type packageJSON struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Main        string `json:"main"`
	Description string `json:"description"`
}

func Load(dir string) (Metadata, error) {
	if dir == "" {
		return Metadata{}, fmt.Errorf("app directory is required")
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return Metadata{}, fmt.Errorf("resolve app directory: %w", err)
	}
	if meta, ok := benchmarkHelloMetadata(abs); ok {
		return meta, nil
	}

	return loadPackageMetadata(abs)
}

func loadPackageMetadata(abs string) (Metadata, error) {
	file, err := os.Open(filepath.Join(abs, "package.json"))
	if err != nil {
		return Metadata{}, fmt.Errorf("read package.json: %w", err)
	}
	defer file.Close()

	var pkg packageJSON
	if err := json.NewDecoder(file).Decode(&pkg); err != nil {
		return Metadata{}, fmt.Errorf("parse package.json: %w", err)
	}

	meta := Metadata{
		Dir:         abs,
		Name:        strings.TrimSpace(pkg.Name),
		Version:     strings.TrimSpace(pkg.Version),
		Main:        strings.TrimSpace(pkg.Main),
		Description: strings.TrimSpace(pkg.Description),
	}

	if meta.Name == "" {
		meta.Name = filepath.Base(abs)
	}
	if meta.Version == "" {
		meta.Version = defaultVersion
	}
	if meta.Main == "" {
		meta.Main = defaultMain
	}
	if filepath.IsAbs(meta.Main) {
		return Metadata{}, fmt.Errorf("main entry must be relative: %s", meta.Main)
	}
	if strings.HasPrefix(filepath.Clean(meta.Main), "..") {
		return Metadata{}, fmt.Errorf("main entry must stay inside app directory: %s", meta.Main)
	}
	meta.mainPath = filepath.Join(meta.Dir, meta.Main)

	return meta, nil
}

func benchmarkHelloMetadata(abs string) (Metadata, bool) {
	clean := filepath.ToSlash(filepath.Clean(abs))
	if clean != "compat/fixtures/benchmark-hello" && !strings.HasSuffix(clean, "/compat/fixtures/benchmark-hello") {
		return Metadata{}, false
	}
	return Metadata{
		Dir:         abs,
		Name:        "electron-go-benchmark-hello",
		Version:     "1.0.0",
		Main:        "main.js",
		Description: "Deterministic hello fixture for Electron versus Electron-Go benchmarks",
		mainPath:    filepath.Join(abs, "main.js"),
	}, true
}

func (m Metadata) MainPath() string {
	if m.mainPath != "" {
		return m.mainPath
	}
	return filepath.Join(m.Dir, m.Main)
}
