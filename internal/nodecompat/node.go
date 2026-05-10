package nodecompat

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	ErrInvalidVersions = errors.New("invalid Node/V8 versions")
	ErrInvalidModule   = errors.New("invalid Node module")
	ErrUnsupportedABI  = errors.New("unsupported native module ABI")
)

type Versions struct {
	Electron string
	Chrome   string
	Node     string
	V8       string
	Modules  string
}

type ModuleKind string

const (
	ModuleCommonJS ModuleKind = "commonjs"
	ModuleESM      ModuleKind = "module"
	ModuleNative   ModuleKind = "native"
)

type ModuleRequest struct {
	Specifier   string
	PackageType ModuleKind
	NativeABI   string
	AllowedABIs []string
}

type ModuleResolution struct {
	Specifier string
	Kind      ModuleKind
	NativeABI string
}

func NormalizeVersions(versions Versions) (Versions, error) {
	versions.Electron = strings.TrimSpace(versions.Electron)
	versions.Chrome = strings.TrimSpace(versions.Chrome)
	versions.Node = strings.TrimSpace(versions.Node)
	versions.V8 = strings.TrimSpace(versions.V8)
	versions.Modules = strings.TrimSpace(versions.Modules)
	if versions.Electron == "" || versions.Chrome == "" || versions.Node == "" || versions.V8 == "" {
		return Versions{}, fmt.Errorf("%w: electron/chrome/node/v8 are required", ErrInvalidVersions)
	}
	return versions, nil
}

func ProcessVersions(versions Versions) (map[string]string, error) {
	normalized, err := NormalizeVersions(versions)
	if err != nil {
		return nil, err
	}
	out := map[string]string{
		"electron": normalized.Electron,
		"chrome":   normalized.Chrome,
		"node":     normalized.Node,
		"v8":       normalized.V8,
	}
	if normalized.Modules != "" {
		out["modules"] = normalized.Modules
	}
	return out, nil
}

func ResolveModule(req ModuleRequest) (ModuleResolution, error) {
	req.Specifier = strings.TrimSpace(req.Specifier)
	if req.Specifier == "" || strings.ContainsAny(req.Specifier, "\x00\r\n") {
		return ModuleResolution{}, fmt.Errorf("%w: specifier", ErrInvalidModule)
	}
	kind := req.PackageType
	if kind == "" {
		kind = inferKind(req.Specifier)
	}
	if kind != ModuleCommonJS && kind != ModuleESM && kind != ModuleNative {
		return ModuleResolution{}, fmt.Errorf("%w: package type", ErrInvalidModule)
	}
	nativeABI := strings.TrimSpace(req.NativeABI)
	if kind == ModuleNative {
		if nativeABI == "" {
			return ModuleResolution{}, fmt.Errorf("%w: native ABI is required", ErrUnsupportedABI)
		}
		if len(req.AllowedABIs) > 0 && !containsABI(req.AllowedABIs, nativeABI) {
			return ModuleResolution{}, fmt.Errorf("%w: %s", ErrUnsupportedABI, nativeABI)
		}
	}
	return ModuleResolution{
		Specifier: req.Specifier,
		Kind:      kind,
		NativeABI: nativeABI,
	}, nil
}

func inferKind(specifier string) ModuleKind {
	switch filepath.Ext(specifier) {
	case ".mjs":
		return ModuleESM
	case ".node":
		return ModuleNative
	default:
		return ModuleCommonJS
	}
}

func containsABI(allowed []string, abi string) bool {
	for _, candidate := range allowed {
		if strings.TrimSpace(candidate) == abi {
			return true
		}
	}
	return false
}
