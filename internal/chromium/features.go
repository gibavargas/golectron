package chromium

import (
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidFeature = errors.New("invalid Chromium feature state")

type SharedTextureFormat string

const (
	TextureRGBA8   SharedTextureFormat = "rgba8"
	TextureBGRA8   SharedTextureFormat = "bgra8"
	TextureRGB10A2 SharedTextureFormat = "rgb10a2"
)

type Capabilities struct {
	ChromiumVersion string
	WebGL           bool
	WebGPU          bool
	PDF             bool
	MediaCapture    bool
	SharedTextures  []SharedTextureFormat
	LOAFAttribution bool
	WasmTrapHandler bool
}

type RendererDiagnostic struct {
	FrameURL string
	Kind     string
	Detail   string
}

type Registry struct {
	capabilities Capabilities
	features     map[string]bool
	diagnostics  []RendererDiagnostic
}

func NewRegistry(capabilities Capabilities) (*Registry, error) {
	normalized, err := normalizeCapabilities(capabilities)
	if err != nil {
		return nil, err
	}
	return &Registry{capabilities: normalized, features: make(map[string]bool)}, nil
}

func (r *Registry) Capabilities() Capabilities {
	out := r.capabilities
	out.SharedTextures = append([]SharedTextureFormat(nil), out.SharedTextures...)
	return out
}

func (r *Registry) SetFeature(name string, enabled bool) error {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, "\x00\r\n") {
		return fmt.Errorf("%w: feature name", ErrInvalidFeature)
	}
	r.features[name] = enabled
	return nil
}

func (r *Registry) FeatureEnabled(name string) bool {
	return r.features[strings.TrimSpace(name)]
}

func (r *Registry) RecordDiagnostic(diagnostic RendererDiagnostic) error {
	diagnostic.FrameURL = strings.TrimSpace(diagnostic.FrameURL)
	diagnostic.Kind = strings.TrimSpace(diagnostic.Kind)
	diagnostic.Detail = strings.TrimSpace(diagnostic.Detail)
	if diagnostic.FrameURL == "" || diagnostic.Kind == "" {
		return fmt.Errorf("%w: diagnostic", ErrInvalidFeature)
	}
	r.diagnostics = append(r.diagnostics, diagnostic)
	return nil
}

func (r *Registry) Diagnostics() []RendererDiagnostic {
	return append([]RendererDiagnostic(nil), r.diagnostics...)
}

func normalizeCapabilities(capabilities Capabilities) (Capabilities, error) {
	capabilities.ChromiumVersion = strings.TrimSpace(capabilities.ChromiumVersion)
	if capabilities.ChromiumVersion == "" {
		return Capabilities{}, fmt.Errorf("%w: chromium version", ErrInvalidFeature)
	}
	seen := make(map[SharedTextureFormat]bool, len(capabilities.SharedTextures))
	for _, format := range capabilities.SharedTextures {
		if format != TextureRGBA8 && format != TextureBGRA8 && format != TextureRGB10A2 {
			return Capabilities{}, fmt.Errorf("%w: texture format", ErrInvalidFeature)
		}
		seen[format] = true
	}
	capabilities.SharedTextures = capabilities.SharedTextures[:0]
	for _, format := range []SharedTextureFormat{TextureRGBA8, TextureBGRA8, TextureRGB10A2} {
		if seen[format] {
			capabilities.SharedTextures = append(capabilities.SharedTextures, format)
		}
	}
	return capabilities, nil
}
