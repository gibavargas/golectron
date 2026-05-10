package chromium

import (
	"errors"
	"reflect"
	"testing"
)

func TestRegistryCapabilities(t *testing.T) {
	registry, err := NewRegistry(Capabilities{
		ChromiumVersion: " 148.0.7778.96 ",
		WebGL:           true,
		WebGPU:          true,
		PDF:             true,
		MediaCapture:    true,
		SharedTextures:  []SharedTextureFormat{TextureRGB10A2, TextureRGBA8, TextureRGBA8},
		LOAFAttribution: true,
		WasmTrapHandler: true,
	})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	capabilities := registry.Capabilities()
	if capabilities.ChromiumVersion != "148.0.7778.96" || !capabilities.WebGL || !capabilities.WebGPU || !capabilities.PDF || !capabilities.MediaCapture || !capabilities.LOAFAttribution || !capabilities.WasmTrapHandler {
		t.Fatalf("Capabilities() = %#v", capabilities)
	}
	wantTextures := []SharedTextureFormat{TextureRGBA8, TextureRGB10A2}
	if !reflect.DeepEqual(capabilities.SharedTextures, wantTextures) {
		t.Fatalf("SharedTextures = %#v, want %#v", capabilities.SharedTextures, wantTextures)
	}
	capabilities.SharedTextures[0] = TextureBGRA8
	if got := registry.Capabilities().SharedTextures[0]; got != TextureRGBA8 {
		t.Fatalf("Capabilities() returned mutable textures: %q", got)
	}
}

func TestRegistryRejectsInvalidCapabilities(t *testing.T) {
	cases := []Capabilities{
		{},
		{ChromiumVersion: "148", SharedTextures: []SharedTextureFormat{"yuv"}},
	}
	for _, tc := range cases {
		if _, err := NewRegistry(tc); !errors.Is(err, ErrInvalidFeature) {
			t.Fatalf("NewRegistry(%#v) error = %v, want ErrInvalidFeature", tc, err)
		}
	}
}

func TestFeatureFlagsAndDiagnostics(t *testing.T) {
	registry, err := NewRegistry(Capabilities{ChromiumVersion: "148.0.7778.96"})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	if err := registry.SetFeature("WebAssemblyTrapHandler", true); err != nil {
		t.Fatalf("SetFeature() error = %v", err)
	}
	if !registry.FeatureEnabled(" WebAssemblyTrapHandler ") {
		t.Fatal("FeatureEnabled() = false, want true")
	}
	if err := registry.SetFeature("bad\nfeature", true); !errors.Is(err, ErrInvalidFeature) {
		t.Fatalf("SetFeature(invalid) error = %v, want ErrInvalidFeature", err)
	}
	if err := registry.RecordDiagnostic(RendererDiagnostic{FrameURL: " https://example.test ", Kind: "loaf", Detail: "long frame"}); err != nil {
		t.Fatalf("RecordDiagnostic() error = %v", err)
	}
	diagnostics := registry.Diagnostics()
	if len(diagnostics) != 1 || diagnostics[0].FrameURL != "https://example.test" || diagnostics[0].Kind != "loaf" {
		t.Fatalf("Diagnostics() = %#v", diagnostics)
	}
	diagnostics[0].Kind = "mutated"
	if got := registry.Diagnostics()[0].Kind; got != "loaf" {
		t.Fatalf("Diagnostics() returned mutable state: %q", got)
	}
	if err := registry.RecordDiagnostic(RendererDiagnostic{Kind: "loaf"}); !errors.Is(err, ErrInvalidFeature) {
		t.Fatalf("RecordDiagnostic(invalid) error = %v, want ErrInvalidFeature", err)
	}
}
