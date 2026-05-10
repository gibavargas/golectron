package nodecompat

import (
	"errors"
	"reflect"
	"testing"
)

func TestProcessVersions(t *testing.T) {
	got, err := ProcessVersions(Versions{
		Electron: " 42.0.0 ",
		Chrome:   "148.0.7778.96",
		Node:     "24.15.0",
		V8:       "14.8.178.14",
		Modules:  "137",
	})
	if err != nil {
		t.Fatalf("ProcessVersions() error = %v", err)
	}
	want := map[string]string{
		"electron": "42.0.0",
		"chrome":   "148.0.7778.96",
		"node":     "24.15.0",
		"v8":       "14.8.178.14",
		"modules":  "137",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ProcessVersions() = %#v, want %#v", got, want)
	}
}

func TestProcessVersionsRejectsMissingRequiredVersions(t *testing.T) {
	if _, err := ProcessVersions(Versions{Electron: "42.0.0", Node: "24.15.0"}); !errors.Is(err, ErrInvalidVersions) {
		t.Fatalf("ProcessVersions(missing) error = %v, want ErrInvalidVersions", err)
	}
}

func TestResolveModuleInfersKinds(t *testing.T) {
	cases := []struct {
		spec string
		want ModuleKind
	}{
		{spec: "main.js", want: ModuleCommonJS},
		{spec: "main.mjs", want: ModuleESM},
		{spec: "addon.node", want: ModuleNative},
	}
	for _, tc := range cases {
		got, err := ResolveModule(ModuleRequest{Specifier: tc.spec, NativeABI: "137"})
		if err != nil {
			t.Fatalf("ResolveModule(%s) error = %v", tc.spec, err)
		}
		if got.Kind != tc.want {
			t.Fatalf("ResolveModule(%s).Kind = %q, want %q", tc.spec, got.Kind, tc.want)
		}
	}
}

func TestResolveModuleValidatesNativeABI(t *testing.T) {
	if _, err := ResolveModule(ModuleRequest{Specifier: "addon.node"}); !errors.Is(err, ErrUnsupportedABI) {
		t.Fatalf("ResolveModule(no ABI) error = %v, want ErrUnsupportedABI", err)
	}
	if _, err := ResolveModule(ModuleRequest{Specifier: "addon.node", NativeABI: "136", AllowedABIs: []string{"137"}}); !errors.Is(err, ErrUnsupportedABI) {
		t.Fatalf("ResolveModule(wrong ABI) error = %v, want ErrUnsupportedABI", err)
	}
	got, err := ResolveModule(ModuleRequest{Specifier: "addon.node", NativeABI: "137", AllowedABIs: []string{"137"}})
	if err != nil {
		t.Fatalf("ResolveModule(valid native) error = %v", err)
	}
	if got.NativeABI != "137" || got.Kind != ModuleNative {
		t.Fatalf("native resolution = %#v", got)
	}
}

func TestResolveModuleRejectsInvalidRequests(t *testing.T) {
	cases := []ModuleRequest{
		{},
		{Specifier: "bad\nspecifier"},
		{Specifier: "main.js", PackageType: "typescript"},
	}
	for _, tc := range cases {
		if _, err := ResolveModule(tc); !errors.Is(err, ErrInvalidModule) {
			t.Fatalf("ResolveModule(%#v) error = %v, want ErrInvalidModule", tc, err)
		}
	}
}
