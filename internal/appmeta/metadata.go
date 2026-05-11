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

	return meta, nil
}

func (m Metadata) MainPath() string {
	return filepath.Join(m.Dir, m.Main)
}
